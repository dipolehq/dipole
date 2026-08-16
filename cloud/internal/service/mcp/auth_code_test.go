package mcp

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAuthCodeStore_CreateAndConsume(t *testing.T) {
	store := NewAuthCodeStore()
	clientID := uuid.New()
	tenantID := uuid.New()
	policyID := uuid.New()

	code := store.Create(clientID, tenantID, "http://localhost:7777/cb", "challenge123", "openid", nil, policyID)
	if len(code) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(code))
	}

	pending, err := store.Consume(code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pending.ClientID != clientID {
		t.Error("client_id mismatch")
	}
	if pending.CodeChallenge != "challenge123" {
		t.Error("code_challenge mismatch")
	}
	if pending.PolicyID != policyID {
		t.Error("policy_id mismatch")
	}
}

func TestAuthCodeStore_SingleUse(t *testing.T) {
	store := NewAuthCodeStore()
	code := store.Create(uuid.New(), uuid.New(), "http://localhost/cb", "ch", "openid", nil, uuid.New())

	_, err := store.Consume(code)
	if err != nil {
		t.Fatalf("first consume should succeed: %v", err)
	}

	_, err = store.Consume(code)
	if err != ErrCodeUsed {
		t.Fatalf("expected ErrCodeUsed, got %v", err)
	}
}

func TestAuthCodeStore_NotFound(t *testing.T) {
	store := NewAuthCodeStore()

	_, err := store.Consume("nonexistent")
	if err != ErrCodeNotFound {
		t.Fatalf("expected ErrCodeNotFound, got %v", err)
	}
}

func TestAuthCodeStore_Expired(t *testing.T) {
	store := NewAuthCodeStore()
	clientID := uuid.New()
	tenantID := uuid.New()

	code := store.Create(clientID, tenantID, "http://localhost/cb", "ch", "openid", nil, uuid.New())

	store.mu.Lock()
	store.codes[code].ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	store.mu.Unlock()

	_, err := store.Consume(code)
	if err != ErrCodeExpired {
		t.Fatalf("expected ErrCodeExpired, got %v", err)
	}
}

func TestAuthCodeStore_SweepExpired_RemovesUsedAndExpired(t *testing.T) {
	store := NewAuthCodeStore()
	clientID := uuid.New()
	tenantID := uuid.New()

	usedCode := store.Create(clientID, tenantID, "http://localhost/cb", "ch1", "openid", nil, uuid.New())
	if _, err := store.Consume(usedCode); err != nil {
		t.Fatalf("consume should succeed: %v", err)
	}

	expiredCode := store.Create(clientID, tenantID, "http://localhost/cb", "ch2", "openid", nil, uuid.New())
	store.mu.Lock()
	store.codes[expiredCode].ExpiresAt = time.Now().UTC().Add(-1 * time.Minute)
	store.mu.Unlock()

	liveCode := store.Create(clientID, tenantID, "http://localhost/cb", "ch3", "openid", nil, uuid.New())

	store.SweepExpired(time.Now().UTC())

	store.mu.RLock()
	_, usedStillPresent := store.codes[usedCode]
	_, expiredStillPresent := store.codes[expiredCode]
	_, liveStillPresent := store.codes[liveCode]
	store.mu.RUnlock()

	if usedStillPresent {
		t.Error("expected used code to be swept")
	}
	if expiredStillPresent {
		t.Error("expected expired code to be swept")
	}
	if !liveStillPresent {
		t.Error("expected live, unused code to remain")
	}
}
