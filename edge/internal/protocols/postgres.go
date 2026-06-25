/*
postgres.go implements the PostgreSQL wire protocol handler for credential
injection. Intercepts the authentication handshake between client and server,
injecting credentials from Cloud without the client knowing. Supports MD5
and cleartext password authentication methods.

PostgreSQL message format: [1-byte type | 4-byte length (includes self) | payload]
Exception: StartupMessage has no type byte (just length + protocol_version + params).
*/
package protocols

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/google/uuid"
)

type PostgresHandler struct {
	Logger *slog.Logger
}

type PGCredential struct {
	Username string
	Password string
}

func (h *PostgresHandler) Handle(clientConn, serverConn net.Conn, cred *PGCredential, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	// Read StartupMessage from client
	startup, err := readPGStartupMessage(clientConn)
	if err != nil {
		h.Logger.Error("pg: read startup message", "error", err, "connection_id", connID)
		return
	}

	// Modify username if credential provides a different one
	if cred.Username != "" {
		startup = replaceStartupUser(startup, cred.Username)
	}

	// Forward StartupMessage to server
	if _, err := serverConn.Write(startup); err != nil {
		h.Logger.Error("pg: forward startup", "error", err, "connection_id", connID)
		return
	}

	// Read auth challenge from server
	msgType, payload, err := readPGMessage(serverConn)
	if err != nil {
		h.Logger.Error("pg: read auth challenge", "error", err, "connection_id", connID)
		return
	}

	if msgType == 'E' {
		// ErrorResponse — forward to client and close
		writePGMessage(clientConn, 'E', payload)
		h.Logger.Error("pg: server sent error during auth", "connection_id", connID)
		return
	}

	if msgType != 'R' || len(payload) < 4 {
		h.Logger.Error("pg: unexpected auth message", "type", string(msgType), "connection_id", connID)
		return
	}

	authMethod := binary.BigEndian.Uint32(payload[:4])
	username := cred.Username
	if username == "" {
		username = extractStartupUser(startup)
	}

	var passwordMsg string
	switch authMethod {
	case 5: // MD5
		if len(payload) < 8 {
			h.Logger.Error("pg: md5 auth missing salt", "connection_id", connID)
			return
		}
		salt := payload[4:8]
		passwordMsg = computePGMD5(username, cred.Password, salt)
	case 3: // Cleartext
		passwordMsg = cred.Password
	case 0: // AuthenticationOk (no auth needed)
		writePGMessage(clientConn, 'R', payload)
		// Forward remaining server messages until ReadyForQuery, then passthrough
		for {
			mt, pl, err := readPGMessage(serverConn)
			if err != nil {
				return
			}
			writePGMessage(clientConn, mt, pl)
			if mt == 'Z' {
				break
			}
		}
		go func() { io.Copy(serverConn, clientConn) }()
		io.Copy(clientConn, serverConn)
		return
	default:
		h.Logger.Error("pg: unsupported auth method", "method", authMethod, "connection_id", connID)
		return
	}

	// Send PasswordMessage to server
	pwBytes := []byte(passwordMsg)
	pwBytes = append(pwBytes, 0) // null terminator
	writePGMessage(serverConn, 'p', pwBytes)

	// Read response (should be AuthenticationOk)
	msgType, payload, err = readPGMessage(serverConn)
	if err != nil {
		h.Logger.Error("pg: read auth response", "error", err, "connection_id", connID)
		return
	}

	if msgType == 'E' {
		writePGMessage(clientConn, 'E', payload)
		h.Logger.Error("pg: auth failed", "connection_id", connID)
		return
	}

	// Forward AuthenticationOk to client
	writePGMessage(clientConn, msgType, payload)

	// Forward any remaining server messages until ReadyForQuery
	for {
		mt, pl, err := readPGMessage(serverConn)
		if err != nil {
			break
		}
		writePGMessage(clientConn, mt, pl)
		if mt == 'Z' { // ReadyForQuery
			break
		}
	}

	// Bidirectional passthrough for all subsequent traffic
	go func() { io.Copy(serverConn, clientConn) }()
	io.Copy(clientConn, serverConn)
}

func computePGMD5(username, password string, salt []byte) string {
	inner := md5.Sum([]byte(password + username))
	innerHex := hex.EncodeToString(inner[:])
	outer := md5.Sum([]byte(innerHex + string(salt)))
	return "md5" + hex.EncodeToString(outer[:])
}

func readPGStartupMessage(conn net.Conn) ([]byte, error) {
	// StartupMessage: [4-byte length] [4-byte protocol] [params...]
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(conn, lenBuf); err != nil {
		return nil, err
	}
	msgLen := int(binary.BigEndian.Uint32(lenBuf))
	if msgLen < 8 || msgLen > 10240 {
		return nil, fmt.Errorf("invalid startup message length: %d", msgLen)
	}

	msg := make([]byte, msgLen)
	copy(msg[:4], lenBuf)
	if _, err := io.ReadFull(conn, msg[4:]); err != nil {
		return nil, err
	}
	return msg, nil
}

func readPGMessage(conn net.Conn) (byte, []byte, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}
	msgType := header[0]
	msgLen := int(binary.BigEndian.Uint32(header[1:5])) - 4
	if msgLen < 0 || msgLen > 1<<20 {
		return 0, nil, fmt.Errorf("invalid message length: %d", msgLen)
	}

	payload := make([]byte, msgLen)
	if msgLen > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}
	return msgType, payload, nil
}

func writePGMessage(conn net.Conn, msgType byte, payload []byte) {
	buf := make([]byte, 5+len(payload))
	buf[0] = msgType
	binary.BigEndian.PutUint32(buf[1:5], uint32(len(payload)+4))
	copy(buf[5:], payload)
	conn.Write(buf)
}

func extractStartupUser(startup []byte) string {
	// Skip length(4) + protocol(4), parse key=value\0 pairs
	params := startup[8:]
	for len(params) > 1 {
		keyEnd := indexOf(params, 0)
		if keyEnd < 0 {
			break
		}
		key := string(params[:keyEnd])
		params = params[keyEnd+1:]

		valEnd := indexOf(params, 0)
		if valEnd < 0 {
			break
		}
		val := string(params[:valEnd])
		params = params[valEnd+1:]

		if key == "user" {
			return val
		}
	}
	return ""
}

func replaceStartupUser(startup []byte, newUser string) []byte {
	// Rebuild StartupMessage with new username
	protocolVersion := startup[4:8]
	params := startup[8:]

	var newParams []byte
	replaced := false
	for len(params) > 1 {
		keyEnd := indexOf(params, 0)
		if keyEnd < 0 {
			break
		}
		key := string(params[:keyEnd])
		params = params[keyEnd+1:]

		valEnd := indexOf(params, 0)
		if valEnd < 0 {
			break
		}
		val := string(params[:valEnd])
		params = params[valEnd+1:]

		if key == "user" {
			val = newUser
			replaced = true
		}
		newParams = append(newParams, []byte(key)...)
		newParams = append(newParams, 0)
		newParams = append(newParams, []byte(val)...)
		newParams = append(newParams, 0)
	}

	if !replaced {
		newParams = append([]byte("user\x00"+newUser+"\x00"), newParams...)
	}
	newParams = append(newParams, 0) // final terminator

	totalLen := 4 + 4 + len(newParams) // length + protocol + params
	msg := make([]byte, totalLen)
	binary.BigEndian.PutUint32(msg[0:4], uint32(totalLen))
	copy(msg[4:8], protocolVersion)
	copy(msg[8:], newParams)
	return msg
}

func indexOf(b []byte, val byte) int {
	for i, v := range b {
		if v == val {
			return i
		}
	}
	return -1
}
