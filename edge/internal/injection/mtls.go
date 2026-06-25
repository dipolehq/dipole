/*
mtls.go provides mTLS client certificate presentation for X.509-SVID
authentication. When the Server Workload requires mTLS, the proxy presents
an X.509-SVID certificate during the TLS handshake. The cert+key come from
the X.509-SVID Credential Provider. For MVP, Cloud returns both in the
credential response. Key rotation is handled by credential TTL expiry.
*/
package injection

import (
	"crypto/tls"
	"fmt"
)

func BuildMTLSConfig(certChainPEM, privateKeyPEM, serverName string) (*tls.Config, error) {
	cert, err := tls.X509KeyPair([]byte(certChainPEM), []byte(privateKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("parse mtls certificate: %w", err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ServerName:   serverName,
	}, nil
}
