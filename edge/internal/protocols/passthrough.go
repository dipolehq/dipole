/*
passthrough.go provides bidirectional TCP forwarding without inspection.

Passthrough dials the target server, then copies bytes in both directions
until either side closes. No TLS decrypt, no protocol parsing, no credential
injection. Used for unrecognized protocols and database traffic until
protocol-specific handlers are built.
*/
package protocols

import (
	"io"
	"log/slog"
	"net"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func Passthrough(clientConn net.Conn, targetHost string, targetPort int, connID uuid.UUID, logger *slog.Logger) {
	serverAddr := net.JoinHostPort(targetHost, strconv.Itoa(targetPort))
	serverConn, err := net.DialTimeout("tcp", serverAddr, 10*time.Second)
	if err != nil {
		logger.Error("passthrough dial failed", "host", targetHost, "port", targetPort, "error", err, "connection_id", connID)
		clientConn.Close()
		return
	}

	logger.Info("passthrough established", "host", targetHost, "port", targetPort, "connection_id", connID)

	done := make(chan struct{}, 2)
	go func() { io.Copy(serverConn, clientConn); done <- struct{}{} }()
	go func() { io.Copy(clientConn, serverConn); done <- struct{}{} }()
	<-done

	clientConn.Close()
	serverConn.Close()
}
