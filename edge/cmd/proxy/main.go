/*
Agent Proxy entry point. Initializes the TLS MITM pipeline (CA → cache → engine),
starts the explicit proxy listener, and handles graceful shutdown.
*/
package main

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/config"
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

	handler := newConnectionHandler(engine, logger)
	listener := proxy.NewExplicitListener(cfg, logger, handler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := listener.Start(ctx); err != nil {
			logger.Error("listener failed", "error", err)
			cancel()
		}
	}()

	waitForShutdown(cancel, cfg, logger, listener)
}

func newConnectionHandler(engine *dipoletls.MITMEngine, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		if targetPort == 443 || targetPort == 8443 {
			handleTLS(engine, clientConn, targetHost, targetPort, connID, logger)
		} else {
			handlePassthrough(clientConn, targetHost, targetPort, connID, logger)
		}
	}
}

func handleTLS(engine *dipoletls.MITMEngine, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	clientPlain, serverConn, err := engine.Decrypt(clientConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	clientReader := bufio.NewReader(clientPlain)

	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if err != io.EOF {
				logger.Debug("read request failed", "host", targetHost, "error", err, "connection_id", connID)
			}
			return
		}

		req.URL.Host = targetHost
		req.URL.Scheme = "https"

		logger.Info("request",
			"method", req.Method,
			"host", targetHost,
			"path", req.URL.Path,
			"connection_id", connID,
		)

		if err := req.Write(serverConn); err != nil {
			logger.Debug("write to server failed", "error", err, "connection_id", connID)
			return
		}

		resp, err := http.ReadResponse(bufio.NewReader(serverConn), req)
		if err != nil {
			logger.Debug("read response failed", "error", err, "connection_id", connID)
			return
		}

		if err := resp.Write(clientPlain); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if req.Close || resp.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
			return
		}
	}
}

func handlePassthrough(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("passthrough dial failed", "host", targetHost, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	logger.Info("passthrough", "host", targetHost, "port", targetPort, "connection_id", connID)

	done := make(chan struct{}, 2)
	go func() { io.Copy(serverConn, clientConn); done <- struct{}{} }()
	go func() { io.Copy(clientConn, serverConn); done <- struct{}{} }()
	<-done

	clientConn.Close()
	serverConn.Close()
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
