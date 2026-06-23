/*
Agent Proxy entry point. Initializes the TLS MITM pipeline, protocol detection,
credential injection, metrics server, and both explicit + transparent listeners.
*/
package main

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/injection"
	"github.com/dipolehq/dipole/edge/internal/metrics"
	"github.com/dipolehq/dipole/edge/internal/protocols"
	"github.com/dipolehq/dipole/edge/internal/proxy"
	dipoletls "github.com/dipolehq/dipole/edge/internal/tls"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	logger.Info("starting agent proxy",
		"tenant_id", cfg.TenantID,
		"port", cfg.HTTPServerPort,
		"sigterm_strategy", cfg.SIGTERMStrategy,
	)

	ca, err := dipoletls.NewCertificateAuthority()
	if err != nil {
		logger.Error("failed to create CA", "error", err)
		os.Exit(1)
	}
	logger.Info("root CA generated")

	cache := dipoletls.NewCertCache(ca)
	engine := dipoletls.NewMITMEngine(cache)
	httpHandler := &protocols.HTTPHandler{Logger: logger}

	handler := newConnectionHandler(engine, httpHandler, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics server
	go func() {
		if err := metrics.StartMetricsServer(cfg.MetricsPort); err != nil {
			logger.Warn("metrics server failed", "error", err)
		}
	}()

	// Start explicit listener
	explicitListener := proxy.NewExplicitListener(cfg, logger, handler)
	go func() {
		if err := explicitListener.Start(ctx); err != nil {
			logger.Error("explicit listener failed", "error", err)
			cancel()
		}
	}()

	// Start transparent listener
	transparentListener := proxy.NewTransparentListener(cfg, logger, handler)
	go func() {
		if err := transparentListener.Start(ctx); err != nil {
			logger.Warn("transparent listener failed (expected on non-Linux)", "error", err)
		}
	}()

	waitForShutdown(cancel, cfg, logger, explicitListener)
}

func newConnectionHandler(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		proto, reader := protocols.Detect(targetHost, targetPort, clientConn)
		logger.Info("protocol detected", "protocol", proto.String(), "host", targetHost, "port", targetPort, "connection_id", connID)
		metrics.ConnectionsActive.WithLabelValues(proto.String()).Inc()
		defer metrics.ConnectionsActive.WithLabelValues(proto.String()).Dec()

		switch proto {
		case protocols.ProtoHTTPS, protocols.ProtoSnowflake:
			handleHTTPS(engine, httpHandler, clientConn, reader, targetHost, targetPort, connID, logger)
		case protocols.ProtoHTTP, protocols.ProtoMCP:
			handleHTTP(httpHandler, clientConn, reader, targetHost, targetPort, connID, logger)
		default:
			protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		}

		metrics.ConnectionsTotal.WithLabelValues(proto.String(), "forwarded").Inc()
	}
}

func handleHTTPS(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	// Wrap conn with buffered reader so peeked bytes from Detect are available to TLS handshake
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	cred, config := getTestCredential(targetHost)
	clientReader := bufio.NewReader(clientPlain)
	httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, cred, config, connID)
}

func handleHTTP(httpHandler *protocols.HTTPHandler, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("HTTP dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}
	defer clientConn.Close()
	defer serverConn.Close()

	cred, config := getTestCredential(targetHost)
	httpHandler.Handle(reader, clientConn, serverConn, targetHost, cred, config, connID)
}

func getTestCredential(targetHost string) (*injection.Credential, *injection.InjectionConfig) {
	if targetHost == "httpbin.org" {
		return &injection.Credential{Type: "bearer", Value: "test-hardcoded-12345"},
			&injection.InjectionConfig{Method: "http_bearer"}
	}
	return nil, nil
}

func waitForShutdown(cancel context.CancelFunc, cfg *config.ProxyConfig, logger *slog.Logger, listener *proxy.ExplicitListener) {
	sigCh := make(chan os.Signal, 1)

	if cfg.SIGTERMStrategy == "sigkill" {
		logger.Info("sigkill strategy: ignoring SIGTERM, awaiting SIGKILL")
		signal.Ignore(syscall.SIGTERM)
		signal.Notify(sigCh, syscall.SIGINT)
	} else {
		signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	}

	sig := <-sigCh
	logger.Info("received signal, shutting down", "signal", sig.String())
	cancel()
	listener.Shutdown()
}

func newLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch level {
	case "error":
		lvl = slog.LevelError
	case "warn":
		lvl = slog.LevelWarn
	case "debug", "trace":
		lvl = slog.LevelDebug
	default:
		lvl = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

// bufferedConn wraps a bufio.Reader + net.Conn so that buffered (peeked)
// bytes are read first, then subsequent reads go to the underlying conn.
type bufferedConn struct {
	r *bufio.Reader
	net.Conn
}

func newBufferedConn(r *bufio.Reader, conn net.Conn) *bufferedConn {
	return &bufferedConn{r: r, Conn: conn}
}

func (bc *bufferedConn) Read(b []byte) (int, error) {
	return bc.r.Read(b)
}
