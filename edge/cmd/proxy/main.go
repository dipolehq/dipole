/*
Agent Proxy entry point. Initializes the TLS MITM pipeline, Cloud gRPC client,
protocol detection, metrics server, and both explicit + transparent listeners.
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

	pb "github.com/dipolehq/dipole/gen/dipole/v1"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
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

	// Connect to Cloud if endpoint configured
	var cloud *cloudclient.CloudClient
	if cfg.CloudEndpoint != "" {
		cloud, err = cloudclient.NewCloudClient(cfg.CloudEndpoint, cfg.TenantID, logger)
		if err != nil {
			logger.Error("failed to connect to cloud", "endpoint", cfg.CloudEndpoint, "error", err)
			os.Exit(1)
		}
		defer cloud.Close()
		logger.Info("cloud client connected", "endpoint", cfg.CloudEndpoint)
	} else {
		logger.Warn("no DIPOLE_CLOUD_GRPC_ENDPOINT configured, running without policy evaluation")
	}

	httpHandler := &protocols.HTTPHandler{Logger: logger}
	handler := newConnectionHandler(engine, httpHandler, cloud, cfg, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := metrics.StartMetricsServer(cfg.MetricsPort); err != nil {
			logger.Warn("metrics server failed", "error", err)
		}
	}()

	explicitListener := proxy.NewExplicitListener(cfg, logger, handler)
	go func() {
		if err := explicitListener.Start(ctx); err != nil {
			logger.Error("explicit listener failed", "error", err)
			cancel()
		}
	}()

	transparentListener := proxy.NewTransparentListener(cfg, logger, handler)
	go func() {
		if err := transparentListener.Start(ctx); err != nil {
			logger.Warn("transparent listener failed (expected on non-Linux)", "error", err)
		}
	}()

	waitForShutdown(cancel, cfg, logger, explicitListener)
}

func newConnectionHandler(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, cfg *config.ProxyConfig, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		proto, reader := protocols.Detect(targetHost, targetPort, clientConn)
		logger.Info("protocol detected", "protocol", proto.String(), "host", targetHost, "port", targetPort, "connection_id", connID)
		metrics.ConnectionsActive.WithLabelValues(proto.String()).Inc()
		defer metrics.ConnectionsActive.WithLabelValues(proto.String()).Dec()

		switch proto {
		case protocols.ProtoHTTPS, protocols.ProtoSnowflake:
			handleHTTPS(engine, httpHandler, cloud, cfg, clientConn, reader, targetHost, targetPort, connID, logger)
		case protocols.ProtoHTTP, protocols.ProtoMCP:
			handleHTTP(httpHandler, cloud, cfg, clientConn, reader, targetHost, targetPort, connID, logger)
		default:
			protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		}

		metrics.ConnectionsTotal.WithLabelValues(proto.String(), "forwarded").Inc()
	}
}

func handleHTTPS(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	allowed, cred, injCfg := evaluatePolicy(cloud, cfg, targetHost, targetPort, connID, logger)
	if !allowed {
		clientPlain.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		return
	}

	clientReader := bufio.NewReader(clientPlain)
	httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, cred, injCfg, connID)
}

func handleHTTP(httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	allowed, cred, injCfg := evaluatePolicy(cloud, cfg, targetHost, targetPort, connID, logger)
	if !allowed {
		clientConn.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		clientConn.Close()
		return
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("HTTP dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}
	defer clientConn.Close()
	defer serverConn.Close()

	httpHandler.Handle(reader, clientConn, serverConn, targetHost, cred, injCfg, connID)
}

func evaluatePolicy(cloud *cloudclient.CloudClient, cfg *config.ProxyConfig, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) (allowed bool, cred *injection.Credential, injCfg *injection.InjectionConfig) {
	if cloud == nil {
		return true, nil, nil
	}

	evidence := &pb.ClientEvidence{
		SourceIp:       "",
		Hostname:       "",
		AembitClientId: cfg.ClientWorkloadID,
		Environment:    make(map[string]string),
	}
	if cfg.K8sNamespace != "" {
		evidence.K8S = &pb.K8SEvidence{
			Namespace: cfg.K8sNamespace,
			PodName:   cfg.K8sPodName,
		}
		evidence.Environment["K8S_NAMESPACE"] = cfg.K8sNamespace
		evidence.Environment["K8S_POD_NAME"] = cfg.K8sPodName
	}
	if cfg.ClientWorkloadID != "" {
		evidence.Environment["CLIENT_WORKLOAD_ID"] = cfg.ClientWorkloadID
	}

	ctx := context.Background()
	result, err := cloud.Evaluate(ctx, evidence, targetHost, targetPort)
	if err != nil {
		if cfg.PassThroughBeforeRegistration {
			logger.Debug("cloud evaluation failed, passthrough mode", "error", err, "connection_id", connID)
			return true, nil, nil
		}
		logger.Warn("cloud evaluation failed", "error", err, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("error").Inc()
		return false, nil, nil
	}

	if !result.Allowed {
		logger.Warn("access denied", "reason", result.DenyReason, "host", targetHost, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("denied").Inc()
		return false, nil, nil
	}

	metrics.PolicyEvaluationsTotal.WithLabelValues("allowed").Inc()

	if result.CredentialValue != "" || result.Username != "" {
		cred = &injection.Credential{
			Type:     result.CredentialType,
			Value:    result.CredentialValue,
			Username: result.Username,
			Password: result.Password,
		}
	}

	if result.InjectionMethod != "" {
		injCfg = &injection.InjectionConfig{
			Method:     result.InjectionMethod,
			HeaderName: result.InjectionHeader,
			Prefix:     result.InjectionPrefix,
			QueryParam: result.InjectionQuery,
		}
	}

	return true, cred, injCfg
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
