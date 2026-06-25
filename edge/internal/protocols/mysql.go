/*
mysql.go implements the MySQL wire protocol handler for credential injection.
Parses HandshakeV10 from the server, computes auth response using
mysql_native_password (SHA1-based XOR), sends HandshakeResponse41, and
handles AuthSwitchRequest. Client never sees authentication.

MySQL packet format: [3-byte length (little-endian) | 1-byte sequence_id | payload]
*/
package protocols

import (
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/google/uuid"
)

type MySQLHandler struct {
	Logger *slog.Logger
}

type MySQLCredential struct {
	Username string
	Password string
	Database string
}

func (h *MySQLHandler) Handle(clientConn, serverConn net.Conn, cred *MySQLCredential, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	// Read HandshakeV10 from server
	_, payload, err := readMySQLPacket(serverConn)
	if err != nil {
		h.Logger.Error("mysql: read handshake", "error", err, "connection_id", connID)
		return
	}

	salt, authPlugin, err := parseHandshakeV10(payload)
	if err != nil {
		h.Logger.Error("mysql: parse handshake", "error", err, "connection_id", connID)
		return
	}

	// Compute auth response
	var authData []byte
	switch authPlugin {
	case "mysql_native_password":
		authData = computeNativePassword(cred.Password, salt)
	default:
		// Default to native password
		authData = computeNativePassword(cred.Password, salt)
	}

	// Send HandshakeResponse41
	response := buildHandshakeResponse41(cred.Username, authData, cred.Database, authPlugin)
	if err := writeMySQLPacket(serverConn, 1, response); err != nil {
		h.Logger.Error("mysql: send response", "error", err, "connection_id", connID)
		return
	}

	// Read server response
	_, respPayload, err := readMySQLPacket(serverConn)
	if err != nil {
		h.Logger.Error("mysql: read auth result", "error", err, "connection_id", connID)
		return
	}

	if len(respPayload) == 0 {
		h.Logger.Error("mysql: empty auth response", "connection_id", connID)
		return
	}

	switch respPayload[0] {
	case 0x00: // OK
		// Auth succeeded — passthrough
	case 0xFE: // AuthSwitchRequest
		if err := h.handleAuthSwitch(serverConn, respPayload, cred.Password); err != nil {
			h.Logger.Error("mysql: auth switch failed", "error", err, "connection_id", connID)
			return
		}
	case 0xFF: // Error
		h.Logger.Error("mysql: auth failed", "connection_id", connID)
		return
	default:
		h.Logger.Error("mysql: unexpected response", "marker", respPayload[0], "connection_id", connID)
		return
	}

	// Bidirectional passthrough
	go func() { io.Copy(serverConn, clientConn) }()
	io.Copy(clientConn, serverConn)
}

func (h *MySQLHandler) handleAuthSwitch(serverConn net.Conn, payload []byte, password string) error {
	// AuthSwitchRequest: [0xFE] [plugin_name\0] [plugin_data]
	if len(payload) < 2 {
		return fmt.Errorf("auth switch payload too short")
	}

	rest := payload[1:]
	pluginEnd := indexOf(rest, 0)
	if pluginEnd < 0 {
		return fmt.Errorf("auth switch missing plugin name")
	}
	pluginName := string(rest[:pluginEnd])
	pluginData := rest[pluginEnd+1:]

	// Remove trailing null from plugin data if present
	if len(pluginData) > 0 && pluginData[len(pluginData)-1] == 0 {
		pluginData = pluginData[:len(pluginData)-1]
	}

	var authData []byte
	switch pluginName {
	case "mysql_native_password":
		authData = computeNativePassword(password, pluginData)
	default:
		authData = computeNativePassword(password, pluginData)
	}

	// Send auth response (sequence 3)
	if err := writeMySQLPacket(serverConn, 3, authData); err != nil {
		return err
	}

	// Read final response
	_, finalPayload, err := readMySQLPacket(serverConn)
	if err != nil {
		return err
	}

	if len(finalPayload) > 0 && finalPayload[0] == 0xFF {
		return fmt.Errorf("auth failed after switch")
	}

	return nil
}

func computeNativePassword(password string, salt []byte) []byte {
	if password == "" {
		return nil
	}

	// SHA1(password)
	hash1 := sha1.Sum([]byte(password))
	// SHA1(SHA1(password))
	hash2 := sha1.Sum(hash1[:])
	// SHA1(salt + SHA1(SHA1(password)))
	h := sha1.New()
	h.Write(salt)
	h.Write(hash2[:])
	hash3 := h.Sum(nil)

	// XOR(SHA1(password), SHA1(salt + SHA1(SHA1(password))))
	result := make([]byte, 20)
	for i := 0; i < 20; i++ {
		result[i] = hash1[i] ^ hash3[i]
	}
	return result
}

func parseHandshakeV10(payload []byte) (salt []byte, authPlugin string, err error) {
	if len(payload) < 4 {
		return nil, "", fmt.Errorf("handshake too short")
	}

	// Skip: protocol version (1) + server version (null-terminated)
	pos := 1
	for pos < len(payload) && payload[pos] != 0 {
		pos++
	}
	pos++ // skip null terminator

	if pos+4 > len(payload) {
		return nil, "", fmt.Errorf("handshake truncated after version")
	}

	// Skip thread ID (4 bytes)
	pos += 4

	// Auth-plugin-data-part-1 (8 bytes)
	if pos+8 > len(payload) {
		return nil, "", fmt.Errorf("handshake truncated at salt1")
	}
	salt1 := payload[pos : pos+8]
	pos += 8

	// Filler (1 byte)
	pos++

	if pos+2 > len(payload) {
		return salt1, "mysql_native_password", nil
	}

	// Capability flags lower (2 bytes)
	pos += 2
	// Charset (1 byte)
	pos++
	// Status flags (2 bytes)
	pos += 2
	// Capability flags upper (2 bytes)
	pos += 2

	if pos+1 > len(payload) {
		return salt1, "mysql_native_password", nil
	}

	// Auth-plugin-data length
	authDataLen := int(payload[pos])
	pos++

	// Reserved (10 bytes)
	pos += 10

	// Auth-plugin-data-part-2
	salt2Len := authDataLen - 8
	if salt2Len < 0 {
		salt2Len = 12 // default
	}
	if pos+salt2Len > len(payload) {
		salt2Len = len(payload) - pos
	}

	salt2 := payload[pos : pos+salt2Len]
	pos += salt2Len

	// Remove trailing null from salt2
	if len(salt2) > 0 && salt2[len(salt2)-1] == 0 {
		salt2 = salt2[:len(salt2)-1]
	}

	fullSalt := append(salt1, salt2...)

	// Auth plugin name (null-terminated)
	if pos < len(payload) {
		pluginEnd := pos
		for pluginEnd < len(payload) && payload[pluginEnd] != 0 {
			pluginEnd++
		}
		authPlugin = string(payload[pos:pluginEnd])
	}
	if authPlugin == "" {
		authPlugin = "mysql_native_password"
	}

	return fullSalt, authPlugin, nil
}

func buildHandshakeResponse41(username string, authData []byte, database, authPlugin string) []byte {
	// Capabilities: CLIENT_PROTOCOL_41 | CLIENT_SECURE_CONNECTION | CLIENT_PLUGIN_AUTH
	caps := uint32(0x00000200 | 0x00008000 | 0x00080000)
	if database != "" {
		caps |= 0x00000008 // CLIENT_CONNECT_WITH_DB
	}

	var buf []byte

	// Capability flags (4 bytes)
	capBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(capBuf, caps)
	buf = append(buf, capBuf...)

	// Max packet size (4 bytes) — 16MB
	maxBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(maxBuf, 16*1024*1024)
	buf = append(buf, maxBuf...)

	// Charset (1 byte) — utf8mb4 = 45
	buf = append(buf, 45)

	// Reserved (23 bytes)
	buf = append(buf, make([]byte, 23)...)

	// Username (null-terminated)
	buf = append(buf, []byte(username)...)
	buf = append(buf, 0)

	// Auth response (length-encoded)
	buf = append(buf, byte(len(authData)))
	buf = append(buf, authData...)

	// Database (null-terminated, if capability set)
	if database != "" {
		buf = append(buf, []byte(database)...)
		buf = append(buf, 0)
	}

	// Auth plugin name (null-terminated)
	buf = append(buf, []byte(authPlugin)...)
	buf = append(buf, 0)

	return buf
}

func readMySQLPacket(conn net.Conn) (seq byte, payload []byte, err error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return 0, nil, err
	}

	length := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
	seq = header[3]

	if length < 0 || length > 1<<24 {
		return 0, nil, fmt.Errorf("invalid mysql packet length: %d", length)
	}

	payload = make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return 0, nil, err
		}
	}
	return seq, payload, nil
}

func writeMySQLPacket(conn net.Conn, seq byte, payload []byte) error {
	length := len(payload)
	header := []byte{byte(length), byte(length >> 8), byte(length >> 16), seq}
	buf := append(header, payload...)
	_, err := conn.Write(buf)
	return err
}
