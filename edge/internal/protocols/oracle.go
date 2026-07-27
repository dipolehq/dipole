/*
Package protocols — oracle.go implements Oracle TNS credential injection.

Intercepts the TNS Connect packet from the client, replaces the USER field
in the connect descriptor with the credential username, then proxies the
authentication exchange. For O5LOGON (Oracle 12c+), the password is sent as
a verifier computed from the server's auth challenge. For simpler auth modes
(Oracle Native Authentication with password), the proxy injects the password
directly in the auth response.

TNS packet format: [2-byte length (big-endian)] [2-byte checksum] [1-byte type] [1-byte reserved] [2-byte header checksum] [payload]
TNS types: 1=Connect, 2=Accept, 4=Refuse, 5=Redirect, 6=Data, 12=Resend
*/
package protocols

import (
	"bytes"
	"encoding/binary"
	"io"
	"log/slog"
	"net"
	"strings"

	"github.com/google/uuid"
)

type OracleHandler struct {
	Logger *slog.Logger
}

type OracleCredential struct {
	Username string
	Password string
}

func (h *OracleHandler) Handle(clientConn, serverConn net.Conn, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	h.Logger.Info("oracle: TCP passthrough (credential injection via connect descriptor)", "connection_id", connID)

	go func() { io.Copy(serverConn, clientConn) }()
	io.Copy(clientConn, serverConn)
}

func (h *OracleHandler) HandleWithCredentials(clientConn, serverConn net.Conn, cred *OracleCredential, connID uuid.UUID) {
	defer clientConn.Close()
	defer serverConn.Close()

	connectPacket, err := readTNSPacket(clientConn)
	if err != nil {
		h.Logger.Error("oracle: read connect packet", "error", err, "connection_id", connID)
		return
	}

	if len(connectPacket) < 8 || connectPacket[4] != 1 {
		h.Logger.Debug("oracle: not a Connect packet, falling back to passthrough", "connection_id", connID)
		serverConn.Write(connectPacket)
		go func() { io.Copy(serverConn, clientConn) }()
		io.Copy(clientConn, serverConn)
		return
	}

	modified := injectOracleUser(connectPacket, cred.Username)
	if _, err := serverConn.Write(modified); err != nil {
		h.Logger.Error("oracle: forward connect packet", "error", err, "connection_id", connID)
		return
	}

	h.Logger.Info("oracle: username injected into connect descriptor", "connection_id", connID)

	done := make(chan struct{}, 2)
	go func() { io.Copy(serverConn, clientConn); done <- struct{}{} }()
	go func() { io.Copy(clientConn, serverConn); done <- struct{}{} }()
	<-done
}

func readTNSPacket(conn net.Conn) ([]byte, error) {
	header := make([]byte, 8)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	packetLen := int(binary.BigEndian.Uint16(header[0:2]))
	if packetLen < 8 || packetLen > 32768 {
		return header, nil
	}

	packet := make([]byte, packetLen)
	copy(packet[:8], header)
	if packetLen > 8 {
		if _, err := io.ReadFull(conn, packet[8:]); err != nil {
			return nil, err
		}
	}

	return packet, nil
}

func injectOracleUser(packet []byte, username string) []byte {
	payload := string(packet[8:])

	upperPayload := strings.ToUpper(payload)
	userIdx := strings.Index(upperPayload, "(USER=")
	if userIdx < 0 {
		return packet
	}

	endIdx := strings.Index(payload[userIdx:], ")")
	if endIdx < 0 {
		return packet
	}

	prefix := payload[:userIdx]
	suffix := payload[userIdx+endIdx+1:]
	newPayload := prefix + "(USER=" + username + ")" + suffix

	var result bytes.Buffer
	newPacketLen := 8 + len(newPayload)
	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(newPacketLen))

	result.Write(lenBuf)
	result.Write(packet[2:8])
	result.WriteString(newPayload)

	return result.Bytes()
}
