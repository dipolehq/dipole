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

type mockRefresherConfigResolver struct {
	config *ConsentConfig
	err    error
}

func (m *mockRefresherConfigResolver) GetConsentConfig(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*ConsentConfig, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.config, nil
}

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
	resolver := &mockRefresherConfigResolver{config: &ConsentConfig{TokenEndpoint: srv.URL}}
	refresher := NewOAuthTokenRefresher(resolver)

	entry, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "old-refresh-token", serverID)
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
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	resolver := &mockRefresherConfigResolver{config: &ConsentConfig{TokenEndpoint: srv.URL}}
	refresher := NewOAuthTokenRefresher(resolver)

	entry, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "keep-this-refresh-token", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry.RefreshToken != "keep-this-refresh-token" {
		t.Errorf("expected old refresh token preserved, got '%s'", entry.RefreshToken)
	}
}

func TestOAuthTokenRefresher_Refresh_NoEndpoint_Error(t *testing.T) {
	resolver := &mockRefresherConfigResolver{config: &ConsentConfig{TokenEndpoint: ""}}
	refresher := NewOAuthTokenRefresher(resolver)

	_, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "token", uuid.New())
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestOAuthTokenRefresher_Refresh_ConfigResolutionFails_Error(t *testing.T) {
	resolver := &mockRefresherConfigResolver{err: ErrNoGatewayURL}
	refresher := NewOAuthTokenRefresher(resolver)

	_, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "token", uuid.New())
	if err == nil {
		t.Fatal("expected error when the token endpoint cannot be resolved")
	}
}

func TestOAuthTokenRefresher_Refresh_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer srv.Close()

	resolver := &mockRefresherConfigResolver{config: &ConsentConfig{TokenEndpoint: srv.URL}}
	refresher := NewOAuthTokenRefresher(resolver)

	_, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "revoked-token", uuid.New())
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestOAuthTokenRefresher_Refresh_DefaultExpiry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "token",
			"expires_in":   0,
		})
	}))
	defer srv.Close()

	resolver := &mockRefresherConfigResolver{config: &ConsentConfig{TokenEndpoint: srv.URL}}
	refresher := NewOAuthTokenRefresher(resolver)

	entry, err := refresher.Refresh(context.Background(), testScope(uuid.New()), "rt", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMin := time.Now().UTC().Add(55 * time.Minute)
	if entry.ExpiresAt.Before(expectedMin) {
		t.Errorf("expected ExpiresAt at least 55 minutes from now, got %v", entry.ExpiresAt)
	}
}
