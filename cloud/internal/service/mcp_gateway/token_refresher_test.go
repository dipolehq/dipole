/*
token_refresher_test.go tests the OAuthTokenRefresher that calls downstream
MCP servers' OAuth token endpoints with grant_type=refresh_token.
*/
package mcp_gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOAuthTokenRefresher_Refresh_Success(t *testing.T) {
	var receivedGrantType string
	var receivedRefreshToken string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected form content type, got '%s'", r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		receivedGrantType = r.FormValue("grant_type")
		receivedRefreshToken = r.FormValue("refresh_token")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"expires_in":    7200,
		})
	}))
	defer srv.Close()

	serverID := uuid.New()
	endpoints := map[uuid.UUID]string{serverID: srv.URL}
	refresher := NewOAuthTokenRefresher(endpoints)

	entry, err := refresher.Refresh(context.Background(), "old-refresh-token", serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedGrantType != "refresh_token" {
		t.Errorf("expected grant_type=refresh_token, got '%s'", receivedGrantType)
	}
	if receivedRefreshToken != "old-refresh-token" {
		t.Errorf("expected refresh_token='old-refresh-token', got '%s'", receivedRefreshToken)
	}
	if entry.AccessToken != "new-access-token" {
		t.Errorf("expected access_token='new-access-token', got '%s'", entry.AccessToken)
	}
	if entry.RefreshToken != "new-refresh-token" {
		t.Errorf("expected refresh_token='new-refresh-token', got '%s'", entry.RefreshToken)
	}
	if entry.ExpiresAt.Before(time.Now().UTC().Add(7100 * time.Second)) {
		t.Error("expected ExpiresAt to be approximately 7200 seconds from now")
	}
}

func TestOAuthTokenRefresher_Refresh_PreservesOldRefreshToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Response does NOT include refresh_token — should preserve the old one
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	serverID := uuid.New()
	endpoints := map[uuid.UUID]string{serverID: srv.URL}
	refresher := NewOAuthTokenRefresher(endpoints)

	entry, err := refresher.Refresh(context.Background(), "keep-this-refresh-token", serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.RefreshToken != "keep-this-refresh-token" {
		t.Errorf("expected old refresh token preserved, got '%s'", entry.RefreshToken)
	}
}

func TestOAuthTokenRefresher_Refresh_NoEndpoint_Error(t *testing.T) {
	serverID := uuid.New()
	otherID := uuid.New()
	endpoints := map[uuid.UUID]string{otherID: "https://other.example.com/token"}
	refresher := NewOAuthTokenRefresher(endpoints)

	_, err := refresher.Refresh(context.Background(), "token", serverID)
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestOAuthTokenRefresher_Refresh_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	serverID := uuid.New()
	endpoints := map[uuid.UUID]string{serverID: srv.URL}
	refresher := NewOAuthTokenRefresher(endpoints)

	_, err := refresher.Refresh(context.Background(), "revoked-token", serverID)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestOAuthTokenRefresher_Refresh_DefaultExpiry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// expires_in is 0 — should default to 1 hour
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"expires_in":   0,
		})
	}))
	defer srv.Close()

	serverID := uuid.New()
	endpoints := map[uuid.UUID]string{serverID: srv.URL}
	refresher := NewOAuthTokenRefresher(endpoints)

	entry, err := refresher.Refresh(context.Background(), "rt", serverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should default to approximately 1 hour from now
	expectedMin := time.Now().UTC().Add(55 * time.Minute)
	if entry.ExpiresAt.Before(expectedMin) {
		t.Errorf("expected ExpiresAt at least 55 minutes from now, got %v", entry.ExpiresAt)
	}
}
