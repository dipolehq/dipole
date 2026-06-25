/*
oracle.go handles Oracle Database connections on port 1521.

Oracle uses TNS (Transparent Network Substrate) with O5LOGON authentication
using 12C password verifiers. This is significantly more complex than
PG/MySQL/Redis auth injection. For MVP, Oracle connections are handled as
TCP passthrough — credential injection for Oracle TNS will be implemented
in a future iteration.

TODO: Implement O5LOGON auth injection for thin JDBC clients.
*/
package protocols

import (
	"io"
	"log/slog"
	"net"

	"github.com/google/uuid"
)

type OracleHandler struct {
	Logger *slog.Logger
}

func (h *OracleHandler) Handle(clientConn, serverConn net.Conn, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	h.Logger.Info("oracle: TCP passthrough (TNS auth injection not yet implemented)", "connection_id", connID)

	go func() { io.Copy(serverConn, clientConn) }()
	io.Copy(clientConn, serverConn)
}
