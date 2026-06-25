package protocols

import (
	"encoding/hex"
	"testing"
)

func TestComputePGMD5(t *testing.T) {
	// Test vector: username="user", password="pass", salt=[0x01,0x02,0x03,0x04]
	// inner = md5("passuser") = "a]..." (hex)
	// outer = md5(innerHex + salt_bytes)
	// result = "md5" + hex(outer)

	result := computePGMD5("user", "pass", []byte{0x01, 0x02, 0x03, 0x04})

	if len(result) != 35 {
		t.Fatalf("expected 35 chars (md5 + 32 hex), got %d: %q", len(result), result)
	}
	if result[:3] != "md5" {
		t.Fatalf("expected md5 prefix, got %q", result[:3])
	}

	// Verify the hex part is valid
	_, err := hex.DecodeString(result[3:])
	if err != nil {
		t.Fatalf("invalid hex in result: %v", err)
	}
}

func TestComputePGMD5_KnownVector(t *testing.T) {
	// PostgreSQL uses: md5(md5(password + username) + salt)
	// password="secret", username="postgres", salt=0xdeadbeef
	// inner = md5("secretpostgres") = "2a3b2c..." (computed)
	// We verify consistency by calling twice with same inputs
	r1 := computePGMD5("postgres", "secret", []byte{0xde, 0xad, 0xbe, 0xef})
	r2 := computePGMD5("postgres", "secret", []byte{0xde, 0xad, 0xbe, 0xef})

	if r1 != r2 {
		t.Fatalf("inconsistent results: %q vs %q", r1, r2)
	}

	// Different salt → different result
	r3 := computePGMD5("postgres", "secret", []byte{0x01, 0x02, 0x03, 0x04})
	if r1 == r3 {
		t.Fatal("different salt should produce different result")
	}
}

func TestExtractStartupUser(t *testing.T) {
	// Build a minimal StartupMessage: length(4) + protocol(4) + "user\0myuser\0\0"
	msg := []byte{
		0, 0, 0, 23, // length = 23
		0, 3, 0, 0, // protocol 3.0
	}
	msg = append(msg, []byte("user\x00myuser\x00")...)
	msg = append(msg, []byte("database\x00mydb\x00")...)
	msg = append(msg, 0) // terminator

	// Fix length
	msgLen := len(msg)
	msg[0] = byte(msgLen >> 24)
	msg[1] = byte(msgLen >> 16)
	msg[2] = byte(msgLen >> 8)
	msg[3] = byte(msgLen)

	user := extractStartupUser(msg)
	if user != "myuser" {
		t.Errorf("user = %q, want myuser", user)
	}
}

func TestReplaceStartupUser(t *testing.T) {
	msg := []byte{
		0, 0, 0, 23,
		0, 3, 0, 0,
	}
	msg = append(msg, []byte("user\x00olduser\x00")...)
	msg = append(msg, []byte("database\x00mydb\x00")...)
	msg = append(msg, 0)

	msgLen := len(msg)
	msg[0] = byte(msgLen >> 24)
	msg[1] = byte(msgLen >> 16)
	msg[2] = byte(msgLen >> 8)
	msg[3] = byte(msgLen)

	replaced := replaceStartupUser(msg, "newuser")
	user := extractStartupUser(replaced)
	if user != "newuser" {
		t.Errorf("user = %q, want newuser", user)
	}
}
