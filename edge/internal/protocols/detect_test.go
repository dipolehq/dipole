package protocols

import (
	"bytes"
	"net"
	"testing"
	"time"
)

type mockConn struct {
	net.Conn
	reader *bytes.Reader
}

func (m *mockConn) Read(b []byte) (int, error)         { return m.reader.Read(b) }
func (m *mockConn) Write(b []byte) (int, error)        { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (m *mockConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func newMockConn(data []byte) net.Conn {
	return &mockConn{reader: bytes.NewReader(data)}
}

func TestDetect_TLSClientHello(t *testing.T) {
	// TLS ClientHello starts with 0x16 0x03
	data := []byte{0x16, 0x03, 0x01, 0x00, 0x05, 0x01, 0x00}
	proto, reader := Detect("example.com", 443, newMockConn(data))

	if proto != ProtoHTTPS {
		t.Errorf("got %v, want ProtoHTTPS", proto)
	}
	if reader == nil {
		t.Fatal("reader is nil")
	}
}

func TestDetect_HTTPMethod(t *testing.T) {
	tests := []struct {
		data []byte
		name string
	}{
		{[]byte("GET / HTTP/1.1\r\n"), "GET"},
		{[]byte("POST /api HTTP/1.1\r\n"), "POST"},
		{[]byte("PUT /data HTTP/1.1\r\n"), "PUT"},
		{[]byte("DELETE /item HTTP/1.1\r\n"), "DELETE"},
		{[]byte("HEAD / HTTP/1.1\r\n"), "HEAD"},
		{[]byte("PATCH /res HTTP/1.1\r\n"), "PATCH"},
		{[]byte("OPTIONS * HTTP/1.1\r\n"), "OPTIONS"},
		{[]byte("CONNECT host:443 HTTP/1.1\r\n"), "CONNECT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proto, _ := Detect("example.com", 80, newMockConn(tt.data))
			if proto != ProtoHTTP {
				t.Errorf("got %v, want ProtoHTTP for %s", proto, tt.name)
			}
		})
	}
}

func TestDetect_PostgreSQL(t *testing.T) {
	data := []byte{0x00, 0x00, 0x00, 0x08, 0x00, 0x03, 0x00, 0x00}
	proto, _ := Detect("db.local", 5432, newMockConn(data))

	if proto != ProtoPostgreSQL {
		t.Errorf("got %v, want ProtoPostgreSQL", proto)
	}
}

func TestDetect_MySQL(t *testing.T) {
	proto, _ := Detect("db.local", 3306, newMockConn([]byte{0x0a}))

	if proto != ProtoMySQL {
		t.Errorf("got %v, want ProtoMySQL", proto)
	}
}

func TestDetect_Redis(t *testing.T) {
	proto, _ := Detect("cache.local", 6379, newMockConn([]byte("*1\r\n$4\r\nPING\r\n")))

	if proto != ProtoRedis {
		t.Errorf("got %v, want ProtoRedis", proto)
	}
}

func TestDetect_Oracle(t *testing.T) {
	proto, _ := Detect("ora.local", 1521, newMockConn([]byte{0x06, 0x00, 0x01, 0x00}))

	if proto != ProtoOracle {
		t.Errorf("got %v, want ProtoOracle", proto)
	}
}

func TestDetect_Redshift(t *testing.T) {
	proto, _ := Detect("redshift.local", 5439, newMockConn([]byte{0x00}))

	if proto != ProtoRedshift {
		t.Errorf("got %v, want ProtoRedshift", proto)
	}
}

func TestDetect_Snowflake(t *testing.T) {
	proto, _ := Detect("account.snowflakecomputing.com", 443, newMockConn([]byte{0x16, 0x03}))

	if proto != ProtoSnowflake {
		t.Errorf("got %v, want ProtoSnowflake", proto)
	}
}

func TestDetect_TCPPassthrough(t *testing.T) {
	// Random bytes on non-standard port
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF, 0x00}
	proto, _ := Detect("unknown.local", 9999, newMockConn(data))

	if proto != ProtoTCPPassthrough {
		t.Errorf("got %v, want ProtoTCPPassthrough", proto)
	}
}

func TestDetect_UnknownBytesOnPort443(t *testing.T) {
	// Non-TLS, non-HTTP bytes on port 443
	data := []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	proto, _ := Detect("weird.local", 443, newMockConn(data))

	if proto != ProtoTCPPassthrough {
		t.Errorf("got %v, want ProtoTCPPassthrough", proto)
	}
}
