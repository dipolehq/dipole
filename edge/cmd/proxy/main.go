/*
Agent Proxy entry point. Initializes the TLS MITM pipeline, Cloud gRPC client,
protocol detection, metrics server, and both explicit + transparent listeners.
*/
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/cloudclient"
	"github.com/dipolehq/dipole/edge/internal/config"
	"github.com/dipolehq/dipole/edge/internal/credcache"
	"github.com/dipolehq/dipole/edge/internal/identity"
	"github.com/dipolehq/dipole/edge/internal/injection"
	"github.com/dipolehq/dipole/edge/internal/metrics"
	"github.com/dipolehq/dipole/edge/internal/protocols"
	"github.com/dipolehq/dipole/edge/internal/proxy"
	dipoletls "github.com/dipolehq/dipole/edge/internal/tls"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var cloud *cloudclient.CloudClient
	if cfg.CloudEndpoint != "" {
		cloud, err = cloudclient.NewCloudClient(cfg.CloudEndpoint, cfg.TenantID, cfg.AgentTrustPath, cfg.GRPCInsecure, logger)
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

	var regResult *cloudclient.RegistrationResult
	if cloud != nil {
		regResult = cloudclient.RunRegistration(ctx, cfg.AgentController, cloud, cfg.PassThroughBeforeRegistration, cfg.PostStartMaxWaitSec, logger)
	}

	ca, err := buildCertificateAuthority(regResult, logger)
	if err != nil {
		logger.Error("failed to create CA", "error", err)
		os.Exit(1)
	}

	writeCAToSharedVolume(ca, logger)

	credentialCache := credcache.NewCredentialCache(60 * time.Second)
	go credentialCache.StartCleanupLoop(ctx, 5*time.Minute)

	cache := dipoletls.NewCertCache(ca)
	engine := dipoletls.NewMITMEngine(cache)

	httpHandler := &protocols.HTTPHandler{Logger: logger}
	gatherer := identity.NewEvidenceGatherer(logger, cfg.ProcessIdentificationEnabled)
	handler := newConnectionHandler(engine, httpHandler, cloud, gatherer, credentialCache, cfg, logger)

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

func buildCertificateAuthority(regResult *cloudclient.RegistrationResult, logger *slog.Logger) (*dipoletls.CertificateAuthority, error) {
	if regResult != nil && regResult.CACertPEM != "" && regResult.CAKeyPEM != "" {
		ca, err := dipoletls.LoadCertificateAuthority([]byte(regResult.CACertPEM), []byte(regResult.CAKeyPEM))
		if err != nil {
			logger.Error("failed to load tenant CA from cloud, falling back to local CA", "error", err)
			return dipoletls.NewCertificateAuthority()
		}
		logger.Info("using tenant CA from cloud")
		return ca, nil
	}

	logger.Info("generating local CA (no tenant CA from cloud)")
	return dipoletls.NewCertificateAuthority()
}

func writeCAToSharedVolume(ca *dipoletls.CertificateAuthority, logger *slog.Logger) {
	const caPath = "/dipole/ca/ca.pem"

	if err := os.MkdirAll("/dipole/ca", 0755); err != nil {
		logger.Warn("cannot create CA directory for trust injection", "path", caPath, "error", err)
		return
	}

	if err := os.WriteFile(caPath, ca.CACertPEM(), 0644); err != nil {
		logger.Warn("cannot write CA cert for trust injection", "path", caPath, "error", err)
		return
	}

	logger.Info("CA certificate written for trust injection", "path", caPath)
}

func newConnectionHandler(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		proto, reader := protocols.Detect(targetHost, targetPort, clientConn)

		sourceIP, sourcePortStr, _ := net.SplitHostPort(clientConn.RemoteAddr().String())
		sourcePort, _ := strconv.Atoi(sourcePortStr)

		logger.Info("protocol detected", "protocol", proto.String(), "host", targetHost, "port", targetPort, "connection_id", connID)
		metrics.ConnectionsActive.WithLabelValues(proto.String()).Inc()
		defer metrics.ConnectionsActive.WithLabelValues(proto.String()).Dec()

		var httpInfo *protocols.HTTPRequestInfo
		var credInjected bool
		var pr *policyResult

		switch proto {
		case protocols.ProtoHTTPS:
			httpInfo, credInjected, pr = handleHTTPS(engine, httpHandler, cloud, gatherer, credCache, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoSnowflake:
			credInjected, pr = handleSnowflake(engine, cloud, gatherer, credCache, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoHTTP, protocols.ProtoMCP:
			httpInfo, credInjected, pr = handleHTTP(httpHandler, cloud, gatherer, credCache, cfg, clientConn, reader, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoPostgreSQL:
			credInjected, pr = handlePostgreSQL(cloud, gatherer, credCache, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoMySQL:
			credInjected, pr = handleMySQL(cloud, gatherer, credCache, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoRedis:
			credInjected, pr = handleRedis(cloud, gatherer, credCache, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		default:
			protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		}

		metrics.ConnectionsTotal.WithLabelValues(proto.String(), "forwarded").Inc()

		if cloud != nil {
			var policyID, clientWorkloadID, serverWorkloadID string
			if pr != nil {
				policyID = pr.PolicyID
				clientWorkloadID = pr.ClientWorkloadID
				serverWorkloadID = pr.ServerWorkloadID
			}
			go emitWorkloadEvent(cloud, proto, targetHost, targetPort, sourceIP, sourcePort, connID, credInjected, httpInfo, cfg.ResourceSetID, policyID, clientWorkloadID, serverWorkloadID, logger)
		}
	}
}

func handleHTTPS(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (*protocols.HTTPRequestInfo, bool, *policyResult) {
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return nil, false, nil
	}
	defer clientPlain.Close()

	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		serverConn.Close()
		clientPlain.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		return nil, false, pr
	}

	if pr.InjectionConfig != nil && pr.InjectionConfig.Method == "mtls_x509" {
		serverConn.Close()

		mtlsConn, mtlsErr := dialMTLS(targetHost, targetPort, pr.Credential, logger, connID)
		if mtlsErr != nil {
			clientPlain.Write([]byte("HTTP/1.1 502 Bad Gateway\r\nContent-Length: 17\r\n\r\nmTLS dial failed\n"))
			return nil, false, pr
		}
		serverConn = mtlsConn
	}
	defer serverConn.Close()

	clientReader := bufio.NewReader(clientPlain)
	info := httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, pr.Credential, pr.InjectionConfig, connID)
	return info, pr.Credential != nil, pr
}

func dialMTLS(targetHost string, targetPort int, cred *injection.Credential, logger *slog.Logger, connID uuid.UUID) (net.Conn, error) {
	if cred == nil || cred.Value == "" || cred.Password == "" {
		return nil, fmt.Errorf("mtls_x509: missing certificate or private key")
	}

	tlsCfg, err := injection.BuildMTLSConfig(cred.Value, cred.Password, targetHost)
	if err != nil {
		logger.Warn("mtls config failed", "error", err, "connection_id", connID)
		return nil, err
	}

	serverAddr := net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", serverAddr, tlsCfg)
	if err != nil {
		logger.Warn("mtls dial failed", "host", targetHost, "error", err, "connection_id", connID)
		return nil, err
	}

	return conn, nil
}

func handleSnowflake(engine *dipoletls.MITMEngine, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (bool, *policyResult) {
	bufferedConn := newBufferedConn(reader, clientConn)
	clientPlain, serverConn, err := engine.Decrypt(bufferedConn, targetHost, targetPort)
	if err != nil {
		logger.Warn("TLS MITM failed", "host", targetHost, "error", err, "connection_id", connID)
		return false, nil
	}
	defer clientPlain.Close()
	defer serverConn.Close()

	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		clientPlain.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		return false, pr
	}

	var snowCred *protocols.SnowflakeCredential
	if pr.Credential != nil && pr.Credential.Value != "" {
		snowCred = &protocols.SnowflakeCredential{
			Username: pr.Credential.Username,
			Token:    pr.Credential.Value,
		}
	}

	if snowCred == nil {
		clientReader := bufio.NewReader(clientPlain)
		httpHandler := &protocols.HTTPHandler{Logger: logger}
		_ = httpHandler.Handle(clientReader, clientPlain, serverConn, targetHost, nil, nil, connID)
		return false, pr
	}

	clientReader := bufio.NewReader(clientPlain)
	sfHandler := &protocols.SnowflakeHandler{Logger: logger}
	sfHandler.Handle(clientReader, clientPlain, serverConn, snowCred, connID)
	return true, pr
}

func handleHTTP(httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, reader *bufio.Reader, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (*protocols.HTTPRequestInfo, bool, *policyResult) {
	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		clientConn.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 14\r\n\r\nAccess Denied\n"))
		clientConn.Close()
		return nil, false, pr
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("HTTP dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return nil, false, pr
	}
	defer clientConn.Close()
	defer serverConn.Close()

	info := httpHandler.Handle(reader, clientConn, serverConn, targetHost, pr.Credential, pr.InjectionConfig, connID)
	return info, pr.Credential != nil, pr
}

func handlePostgreSQL(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (bool, *policyResult) {
	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		clientConn.Close()
		return false, pr
	}

	if pr.Credential == nil || (pr.Credential.Username == "" && pr.Credential.Password == "") {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return false, pr
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("postgres dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return false, pr
	}

	pgHandler := &protocols.PostgresHandler{Logger: logger}
	pgHandler.Handle(clientConn, serverConn, &protocols.PGCredential{
		Username: pr.Credential.Username,
		Password: pr.Credential.Password,
	}, connID)
	return true, pr
}

func handleMySQL(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (bool, *policyResult) {
	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		clientConn.Close()
		return false, pr
	}

	if pr.Credential == nil || (pr.Credential.Username == "" && pr.Credential.Password == "") {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return false, pr
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("mysql dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return false, pr
	}

	mysqlHandler := &protocols.MySQLHandler{Logger: logger}
	mysqlHandler.Handle(clientConn, serverConn, &protocols.MySQLCredential{
		Username: pr.Credential.Username,
		Password: pr.Credential.Password,
	}, connID)
	return true, pr
}

func handleRedis(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, sourceIP string, sourcePort int, logger *slog.Logger) (bool, *policyResult) {
	pr := evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
	if !pr.Allowed {
		clientConn.Close()
		return false, pr
	}

	if pr.Credential == nil || pr.Credential.Password == "" {
		protocols.Passthrough(clientConn, targetHost, targetPort, connID, logger)
		return false, pr
	}

	serverConn, err := net.Dial("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
	if err != nil {
		logger.Error("redis dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return false, pr
	}

	redisHandler := &protocols.RedisHandler{Logger: logger}
	redisHandler.Handle(clientConn, serverConn, &protocols.RedisCredential{
		Username: pr.Credential.Username,
		Password: pr.Credential.Password,
	}, connID)
	return true, pr
}

type policyResult struct {
	Allowed          bool
	Credential       *injection.Credential
	InjectionConfig  *injection.InjectionConfig
	PolicyID         string
	ClientWorkloadID string
	ServerWorkloadID string
}

func evaluatePolicy(cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, sourceIP string, sourcePort int, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) *policyResult {
	if cloud == nil {
		return &policyResult{Allowed: true}
	}

	if entry, ok := credCache.Get(targetHost, targetPort); ok {
		metrics.CredentialCacheHits.Inc()
		return &policyResult{
			Allowed:          entry.Allowed,
			Credential:       entry.Credential,
			InjectionConfig:  entry.InjectionConfig,
			PolicyID:         entry.PolicyID,
			ClientWorkloadID: entry.ClientWorkloadID,
			ServerWorkloadID: entry.ServerWorkloadID,
		}
	}

	evidence := gatherer.Gather(sourceIP, sourcePort)

	ctx := context.Background()
	start := time.Now()
	result, err := cloud.Evaluate(ctx, evidence, targetHost, targetPort)
	metrics.PolicyEvaluationDuration.Observe(time.Since(start).Seconds())

	if err != nil {
		if entry, ok := credCache.Get(targetHost, targetPort); ok {
			logger.Debug("serving cached credential during cloud outage", "host", targetHost, "connection_id", connID)
			return &policyResult{
				Allowed:          entry.Allowed,
				Credential:       entry.Credential,
				InjectionConfig:  entry.InjectionConfig,
				PolicyID:         entry.PolicyID,
				ClientWorkloadID: entry.ClientWorkloadID,
				ServerWorkloadID: entry.ServerWorkloadID,
			}
		}
		if cfg.PassThroughBeforeRegistration {
			logger.Debug("cloud evaluation failed, passthrough mode", "error", err, "connection_id", connID)
			return &policyResult{Allowed: true}
		}
		logger.Warn("cloud evaluation failed", "error", err, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("error").Inc()
		return &policyResult{Allowed: false}
	}

	if !result.Allowed {
		logger.Warn("access denied", "reason", result.DenyReason, "host", targetHost, "connection_id", connID)
		metrics.PolicyEvaluationsTotal.WithLabelValues("denied").Inc()
		return &policyResult{
			Allowed:          false,
			PolicyID:         result.PolicyID,
			ClientWorkloadID: result.ClientWorkloadID,
			ServerWorkloadID: result.ServerWorkloadID,
		}
	}

	metrics.PolicyEvaluationsTotal.WithLabelValues("allowed").Inc()
	metrics.CredentialCacheMisses.Inc()

	pr := &policyResult{
		Allowed:          true,
		PolicyID:         result.PolicyID,
		ClientWorkloadID: result.ClientWorkloadID,
		ServerWorkloadID: result.ServerWorkloadID,
	}

	if result.CredentialValue != "" || result.Username != "" || result.AwsAccessKeyId != "" {
		pr.Credential = &injection.Credential{
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
		pr.InjectionConfig = &injection.InjectionConfig{
			Method:     result.InjectionMethod,
			HeaderName: result.InjectionHeader,
			Prefix:     result.InjectionPrefix,
			QueryParam: result.InjectionQuery,
		}
	}

	var expiresAt time.Time
	if result.ExpiresAt > 0 {
		expiresAt = time.Unix(result.ExpiresAt, 0)
	} else {
		expiresAt = time.Now().Add(credCache.DefaultTTL())
	}

	credCache.Set(targetHost, targetPort, &credcache.CacheEntry{
		Allowed:          true,
		Credential:       pr.Credential,
		InjectionConfig:  pr.InjectionConfig,
		PolicyID:         pr.PolicyID,
		ClientWorkloadID: pr.ClientWorkloadID,
		ServerWorkloadID: pr.ServerWorkloadID,
		ExpiresAt:        expiresAt,
	})
	metrics.CredentialCacheSize.Set(float64(credCache.Size()))

	return pr
}

func emitWorkloadEvent(cloud *cloudclient.CloudClient, proto protocols.Protocol, targetHost string, targetPort int, sourceIP string, sourcePort int, connID uuid.UUID, credInjected bool, httpInfo *protocols.HTTPRequestInfo, resourceSetID string, policyID string, clientWorkloadID string, serverWorkloadID string, logger *slog.Logger) {
	eventType := protocolToEventType(proto)

	outcome := "Passthrough"
	if credInjected {
		outcome = "Modified"
	}
	if httpInfo != nil && httpInfo.Injected {
		outcome = "Modified"
	}

	data := &protocols.EventData{
		EventType:        eventType,
		ConnectionID:     connID.String(),
		PolicyID:         policyID,
		ClientWorkloadID: clientWorkloadID,
		ServerWorkloadID: serverWorkloadID,
		ResourceSetID:    resourceSetID,
		Outcome:          outcome,
		TargetHost:       targetHost,
		TargetPort:       targetPort,
		SourceIP:         sourceIP,
		SourcePort:       sourcePort,
	}

	if httpInfo != nil {
		data.HTTPMethod = httpInfo.Method
		data.HTTPUri = httpInfo.URI
		data.HTTPStatus = httpInfo.StatusCode
		data.Headers = httpInfo.Headers
	}

	event := protocols.BuildWorkloadEvent(data)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cloud.IngestEvents(ctx, []*pb.WorkloadEvent{event}); err != nil {
		logger.Debug("workload event emission failed", "error", err, "connection_id", connID)
	}
}

func protocolToEventType(proto protocols.Protocol) string {
	switch proto {
	case protocols.ProtoHTTPS, protocols.ProtoHTTP, protocols.ProtoMCP:
		return "http.request"
	case protocols.ProtoPostgreSQL:
		return "postgres.request"
	case protocols.ProtoMySQL:
		return "mysql.request"
	case protocols.ProtoRedis:
		return "redis.request"
	case protocols.ProtoSnowflake:
		return "snowflake.request"
	default:
		return "tcp.open"
	}
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
