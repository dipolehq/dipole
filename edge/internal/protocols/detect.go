/*
Package protocols provides application protocol detection and handlers.

Detect identifies the protocol of a connection based on destination port and
the first bytes of the stream (via bufio.Reader.Peek). Bytes are never consumed
— the returned reader replays them for the protocol handler.
*/
package protocols

import (
	"bufio"
	"net"
	"strings"
)

type Protocol int

const (
	ProtoHTTPS Protocol = iota
	ProtoHTTP
	ProtoPostgreSQL
	ProtoMySQL
	ProtoRedis
	ProtoOracle
	ProtoSnowflake
	ProtoRedshift
	ProtoMCP
	ProtoTCPPassthrough
)

func (p Protocol) String() string {
	switch p {
	case ProtoHTTPS:
		return "HTTPS"
	case ProtoHTTP:
		return "HTTP"
	case ProtoPostgreSQL:
		return "PostgreSQL"
	case ProtoMySQL:
		return "MySQL"
	case ProtoRedis:
		return "Redis"
	case ProtoOracle:
		return "Oracle"
	case ProtoSnowflake:
		return "Snowflake"
	case ProtoRedshift:
		return "Redshift"
	case ProtoMCP:
		return "MCP"
	default:
		return "TCP"
	}
}

func Detect(targetHost string, targetPort int, conn net.Conn) (Protocol, *bufio.Reader) {
	// Hostname-based detection (before port/bytes)
	if strings.Contains(targetHost, "snowflakecomputing.com") {
		return ProtoSnowflake, bufio.NewReader(conn)
	}

	// Port-based detection for database protocols
	switch targetPort {
	case 5432:
		return ProtoPostgreSQL, bufio.NewReader(conn)
	case 3306:
		return ProtoMySQL, bufio.NewReader(conn)
	case 6379:
		return ProtoRedis, bufio.NewReader(conn)
	case 1521:
		return ProtoOracle, bufio.NewReader(conn)
	case 5439:
		return ProtoRedshift, bufio.NewReader(conn)
	}

	// For HTTP/HTTPS ports, peek at first bytes
	reader := bufio.NewReader(conn)
	peeked, err := reader.Peek(5)
	if err != nil {
		return ProtoTCPPassthrough, reader
	}

	// TLS ClientHello: first byte 0x16 (handshake), second byte 0x03 (TLS version)
	if peeked[0] == 0x16 && peeked[1] == 0x03 {
		return ProtoHTTPS, reader
	}

	// HTTP method detection
	if isHTTPMethod(peeked) {
		return ProtoHTTP, reader
	}

	return ProtoTCPPassthrough, reader
}

func isHTTPMethod(b []byte) bool {
	prefix := string(b[:4])
	switch {
	case prefix == "GET " || prefix == "POST" || prefix == "PUT " ||
		prefix == "HEAD" || prefix == "DELE" || prefix == "PATC" ||
		prefix == "OPTI" || prefix == "CONN":
		return true
	}
	return false
}
