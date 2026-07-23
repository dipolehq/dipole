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

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/identity"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if cloud != nil && cfg.ResourceSetID != "" {
		cloud.SetResourceSetID(cfg.ResourceSetID)
	}

	if cloud != nil {
		if cfg.PassThroughBeforeRegistration {
			go cloudclient.RunRegistration(ctx, cfg.AgentController, cloud, true, cfg.PostStartMaxWaitSec, logger)
		} else {
			cloudclient.RunRegistration(ctx, cfg.AgentController, cloud, false, cfg.PostStartMaxWaitSec, logger)
		}
	}

	httpHandler := &protocols.HTTPHandler{Logger: logger}
	gatherer := identity.NewEvidenceGatherer(logger, cfg.ProcessIdentificationEnabled)
	handler := newConnectionHandler(engine, httpHandler, cloud, gatherer, cfg, logger)

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

func newConnectionHandler(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		proto, reader := protocols.Detect(targetHost, targetPort, clientConn)

		sourceIP, sourcePortStr, _ := net.SplitHostPort(clientConn.RemoteAddr().String())
		sourcePort, _ := strconv.Atoi(sourcePortStr)

		logger.Info("protocol detected", "protocol", proto.String(), "host", targetHost, "port", targetPort, "connection_id", connID)
		metrics.ConnectionsActive.WithLabelValues(proto.String()).Inc()
		defer metrics.ConnectionsActive.WithLabelValues(proto.String()).Dec()

		switch proto {
		case protocols.ProtoHTTPS:
			handleHTTPS(engine, httpHandler, cloud, gatherer, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoSnowflake:
			handleSnowflake(engine, cloud, gatherer, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoHTTP, protocols.ProtoMCP:
			handleHTTP(httpHandler, cloud, gatherer, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoPostgreSQL:
			handlePostgreSQL(cloud, gatherer, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoMySQL:
			handleMySQL(cloud, gatherer, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoRedis:
			handleRedis(cloud, gatherer, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		default:
			protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		}

		metrics.ConnectionsTotal.WithLabelValues(proto.String(), "forwarded").Inc()
	}
}

func handleHTTPS(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	allowed, cred, injCfg := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !allowed {
		clientPlain.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		return
	}

	clientReader := bufio.NewReader(clientPlain)
	httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, cred, injCfg, connID)
}

func handleSnowflake(engine *dipoletls.MITMEngine, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	allowed, cred, _ := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !allowed {
		clientPlain.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		return
	}

	var snowCred *protocols.SnowflakeCredential
	if cred != nil && cred.Value != "" {
		snowCred = &protocols.SnowflakeCredential{
			Username: cred.Username,
			Token:    cred.Value,
		}
	}

	if snowCred == nil {
		clientReader := bufio.NewReader(clientPlain)
		httpHandler := &protocols.HTTPHandler{Logger: logger}
		httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, nil, nil, connID)
		return
	}

	clientReader := bufio.NewReader(clientPlain)
	sfHandler := &protocols.SnowflakeHandler{Logger: logger}
	sfHandler.Handle(clientReader, clientPlain, serverConn, snowCred, connID)
}

func handleHTTP(httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	allowed, cred, injCfg := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
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

func handlePostgreSQL(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	allowed, cred, _ := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !allowed {
		clientConn.Close()
		return
	}

	if cred == nil || (cred.Username == "" && cred.Password == "") {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("postgres dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	pgHandler := &protocols.PostgresHandler{Logger: logger}
	pgHandler.Handle(clientConn, serverConn, &protocols.PGCredential{
		Username: cred.Username,
		Password: cred.Password,
	}, connID)
}

func handleMySQL(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	allowed, cred, _ := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !allowed {
		clientConn.Close()
		return
	}

	if cred == nil || (cred.Username == "" && cred.Password == "") {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("mysql dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	mysqlHandler := &protocols.MySQLHandler{Logger: logger}
	mysqlHandler.Handle(clientConn, serverConn, &protocols.MySQLCredential{
		Username: cred.Username,
		Password: cred.Password,
	}, connID)
}

func handleRedis(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) {
	allowed, cred, _ := evaluatePolicy(cloud, gatherer, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !allowed {
		clientConn.Close()
		return
	}

	if cred == nil || cred.Password == "" {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("redis dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	redisHandler := &protocols.RedisHandler{Logger: logger}
	redisHandler.Handle(clientConn, serverConn, &protocols.RedisCredential{
		Username: cred.Username,
		Password: cred.Password,
	}, connID)
}

func evaluatePolicy(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, cfg *config.ProxyConfig, sourceIP string, sourcePort int, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) (allowed bool, cred *injection.Credential, injCfg *injection.InjectionConfig) {
	if cloud == nil {
		return true, nil, nil
	}

	evidence := gatherer.Gather(sourceIP, sourcePort)

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

	if result.CredentialValue != "" || result.Username != "" || result.AwsAccessKeyId != "" {
		cred = &injection.Credential{
			Type:               result.CredentialType,
			Value:              result.CredentialValue,
			Username:           result.Username,
			Password:           result.Password,
			AwsAccessKeyId:     result.AwsAccessKeyId,
			AwsSecretAccessKey: result.AwsSecretAccessKey,
			AwsSessionToken:    result.AwsSessionToken,
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
