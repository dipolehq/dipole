package events

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactPayload_StripsSensitiveHeaders(t *testing.T) {
	payload := json.RawMessage(`{"application":{"http":{"headers":[{"content-type":"application/json"},{"authorization":"Bearer secret"},{"set-cookie":"session=abc"},{"cookie":"session_id=s3cr3t"},{"x-api-key":"sk_live_key123"}]}}}`)

	redacted := redactPayload(payload)
	stored := string(redacted)

	if !strings.Contains(stored, "content-type") {
		t.Error("content-type should be preserved")
	}
	if strings.Contains(stored, "Bearer secret") {
		t.Error("authorization header should be redacted")
	}
	if strings.Contains(stored, "session=abc") {
		t.Error("set-cookie header should be redacted")
	}
	if strings.Contains(stored, "s3cr3t") {
		t.Error("cookie header should be redacted")
	}
	if strings.Contains(stored, "sk_live_key123") {
		t.Error("x-api-key header should be redacted")
	}
	if !strings.Contains(stored, "[REDACTED]") {
		t.Error("expected [REDACTED] placeholder")
	}
}

func TestRedactPayload_PreservesNonSensitivePayload(t *testing.T) {
	payload := json.RawMessage(`{"application":{"http":{"method":"GET","path":"/api/users"}}}`)
	redacted := redactPayload(payload)
	if string(redacted) != string(payload) {
		t.Errorf("non-sensitive payload should be unchanged, got %s", string(redacted))
	}
}

func TestRedactPayload_HandlesEmptyPayload(t *testing.T) {
	redacted := redactPayload(nil)
	if redacted != nil {
		t.Error("nil payload should return nil")
	}

	redacted = redactPayload(json.RawMessage(`{}`))
	if string(redacted) != `{}` {
		t.Errorf("empty object should be unchanged, got %s", string(redacted))
	}
}
