package mcp_gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAESSharedOAuthTokenResolver_CachedAccessToken_NoExchange(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	cfg := map[string]string{
		"token_url":     "http://unreachable.invalid/token",
		"client_id":     "client-1",
		"client_secret": "secret-1",
		"refresh_token": "refresh-1",
		"access_token":  "still-valid-token",
		"expires_at":    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	plaintext, _ := json.Marshal(cfg)
	encrypted, err := testEncrypt(plaintext, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	resolver := NewAESSharedOAuthTokenResolver(encKey)

	token, err := resolver.Resolve(context.Background(), encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "still-valid-token" {
		t.Errorf("expected cached access token to be returned without exchange, got %q", token)
	}
}

func TestAESSharedOAuthTokenResolver_ExpiredAccessToken_RefreshesAndReturnsNewToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"refreshed-token"}`)
	}))
	defer srv.Close()

	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}

	cfg := map[string]string{
		"token_url":     srv.URL,
		"client_id":     "client-1",
		"client_secret": "secret-1",
		"refresh_token": "refresh-1",
	}
	plaintext, _ := json.Marshal(cfg)
	encrypted, err := testEncrypt(plaintext, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	resolver := NewAESSharedOAuthTokenResolver(encKey)

	token, err := resolver.Resolve(context.Background(), encrypted)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "refreshed-token" {
		t.Errorf("expected refreshed-token, got %q", token)
	}
}

func TestAESSharedOAuthTokenResolver_NoRefreshToken_Error(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 2)
	}

	cfg := map[string]string{
		"token_url": "http://unreachable.invalid/token",
		"client_id": "client-1",
	}
	plaintext, _ := json.Marshal(cfg)
	encrypted, err := testEncrypt(plaintext, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	resolver := NewAESSharedOAuthTokenResolver(encKey)

	_, err = resolver.Resolve(context.Background(), encrypted)
	if err == nil {
		t.Fatal("expected error when no refresh token and no valid cached access token are available")
	}
}

func TestAESSharedOAuthTokenResolver_EmptyConfig_Error(t *testing.T) {
	resolver := NewAESSharedOAuthTokenResolver(make([]byte, 32))

	_, err := resolver.Resolve(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty credential config")
	}
}
