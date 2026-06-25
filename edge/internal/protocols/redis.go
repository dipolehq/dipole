/*
redis.go implements the Redis RESP protocol handler for credential injection.
After TCP connection to the server, sends an AUTH command using credentials
from Cloud before relaying any client commands. Supports both Redis < 6
(password-only) and Redis 6+ ACL (username + password).

RESP format: Arrays as *<count>\r\n followed by bulk strings $<len>\r\n<data>\r\n
*/
package protocols

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/google/uuid"
)

type RedisHandler struct {
	Logger *slog.Logger
}

type RedisCredential struct {
	Username string
	Password string
}

func (h *RedisHandler) Handle(clientConn, serverConn net.Conn, cred *RedisCredential, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	// Send AUTH command to server
	authCmd := buildRedisAuth(cred.Username, cred.Password)
	if _, err := serverConn.Write([]byte(authCmd)); err != nil {
		h.Logger.Error("redis: send AUTH", "error", err, "connection_id", connID)
		return
	}

	// Read server response
	reader := bufio.NewReader(serverConn)
	line, err := reader.ReadString('\n')
	if err != nil {
		h.Logger.Error("redis: read AUTH response", "error", err, "connection_id", connID)
		return
	}

	if len(line) < 3 || line[0] != '+' {
		h.Logger.Error("redis: AUTH failed", "response", line, "connection_id", connID)
		return
	}

	// Auth succeeded — bidirectional passthrough
	// Use the buffered reader for server→client (may have buffered data)
	go func() { io.Copy(serverConn, clientConn) }()
	io.Copy(clientConn, reader)
}

func buildRedisAuth(username, password string) string {
	if username != "" {
		// Redis 6+ ACL: AUTH <username> <password>
		return fmt.Sprintf("*3\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
			len(username), username, len(password), password)
	}
	// Redis < 6: AUTH <password>
	return fmt.Sprintf("*2\r\n$4\r\nAUTH\r\n$%d\r\n%s\r\n",
		len(password), password)
}
