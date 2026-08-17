/*
decrypt.go implements the TLS MITM engine for the Agent Proxy.

MITMEngine terminates the client's TLS using a generated leaf certificate,
then establishes a separate TLS connection to the real server. Returns both
plaintext streams for HTTP inspection. On any error, both sides are closed —
never returns with one connection open and the other closed.
*/
package tls

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"time"
)

type MITMEngine struct {
	certCache *CertCache
}

func NewMITMEngine(cache *CertCache) *MITMEngine {
	return &MITMEngine{certCache: cache}
}

func (m *MITMEngine) Decrypt(ctx context.Context, clientConn net.Conn, targetHost string, targetPort int) (net.Conn, net.Conn, error) {
	cert, err := m.certCache.GetOrGenerate(ctx, targetHost)
	if err != nil {
		clientConn.Close()
		return nil, nil, fmt.Errorf("get cert for %s: %w", targetHost, err)
	}

	tlsClientConn := tls.Server(clientConn, &tls.Config{
		Certificates: []tls.Certificate{*cert},
	})
	if err := tlsClientConn.Handshake(); err != nil {
		tlsClientConn.Close()
		return nil, nil, fmt.Errorf("client TLS handshake for %s: %w", targetHost, err)
	}

	serverAddr := net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
	serverRawConn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		tlsClientConn.Close()
		return nil, nil, fmt.Errorf("dial server %s: %w", serverAddr, err)
	}

	tlsServerConn := tls.Client(serverRawConn, &tls.Config{
		ServerName: targetHost,
	})
	if err := tlsServerConn.Handshake(); err != nil {
		tlsServerConn.Close()
		tlsClientConn.Close()
		return nil, nil, fmt.Errorf("server TLS handshake for %s: %w", serverAddr, err)
	}

	return tlsClientConn, tlsServerConn, nil
}
