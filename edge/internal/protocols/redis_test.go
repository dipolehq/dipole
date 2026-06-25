package protocols

import (
	"testing"
)

func TestBuildRedisAuth_PasswordOnly(t *testing.T) {
	result := buildRedisAuth("", "mypassword")
	expected := "*2\r\n$4\r\nAUTH\r\n$10\r\nmypassword\r\n"

	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}

func TestBuildRedisAuth_UsernameAndPassword(t *testing.T) {
	result := buildRedisAuth("admin", "secret")
	expected := "*3\r\n$4\r\nAUTH\r\n$5\r\nadmin\r\n$6\r\nsecret\r\n"

	if result != expected {
		t.Errorf("got %q, want %q", result, expected)
	}
}
