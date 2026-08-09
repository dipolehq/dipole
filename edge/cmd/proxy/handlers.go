/* Protocol-specific connection handlers for the agent proxy. */
package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"sync/atomic"
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
)

func newConnectionHandler(engine *dipoletls.MITMEngine, httpHandler *protocols.HTTPHandler, cloud *cloudclient.CloudClient, gatherer *identity.EvidenceGatherer, credCache *credcache.CredentialCache, cfg *config.ProxyConfig, activeConnCount *atomic.Int64, logger *slog.Logger) proxy.ConnectionHandler {
	return func(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID) {
		proto, reader := protocols.Detect(targetHost, targetPort, clientConn)

		sourceIP, sourcePortStr, _ := net.SplitHostPort(clientConn.RemoteAddr().String())
		sourcePort, _ := strconv.Atoi(sourcePortStr)
		_, proxyPortStr, _ := net.SplitHostPort(clientConn.LocalAddr().String())
		proxyPort, _ := strconv.Atoi(proxyPortStr)

		logger.Info("protocol detected", "protocol", proto.String(), "host", targetHost, "port", targetPort, "connection_id", connID)
		metrics.ConnectionsActive.WithLabelValues(proto.String()).Inc()
		activeConnCount.Add(1)
		defer metrics.ConnectionsActive.WithLabelValues(proto.String()).Dec()
		defer activeConnCount.Add(-1)

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
		case protocols.ProtoRedshift:
			credInjected, pr = handlePostgreSQL(cloud, gatherer, credCache, cfg, clientConn, targetHost, targetPort, connID, sourceIP, sourcePort, logger)
		case protocols.ProtoOracle:
			pr = evaluatePolicy(cloud, gatherer, credCache, cfg, sourceIP, sourcePort, targetHost, targetPort, connID, logger)
			if !pr.Allowed {
				clientConn.Close()
			} else {
				serverConn, dialErr := net.DialTimeout("tcp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)), 10*time.Second)
				if dialErr != nil {
					logger.Error("oracle dial failed", "host", targetHost, "port", targetPort, "error", dialErr, "connection_id", connID)
					clientConn.Close()
				} else {
					oracleHandler := &protocols.OracleHandler{Logger: logger}
					if pr.Credential != nil && pr.Credential.Username != "" {
						oracleHandler.HandleWithCredentials(clientConn, serverConn, &protocols.OracleCredential{
							Username: pr.Credential.Username,
							Password: pr.Credential.Password,
						}, connID)
						credInjected = true
					} else {
						oracleHandler.Handle(clientConn, serverConn, connID)
					}
				}
			}
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
			go emitWorkloadEvent(cloud, proto, targetHost, targetPort, sourceIP, sourcePort, proxyPort, connID, credInjected, httpInfo, cfg.ResourceSetID, policyID, clientWorkloadID, serverWorkloadID, logger)
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
