package mcp_gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockConsentTokenStore struct {
	stored               *UserTokenEntry
	key                  string
	credentialProviderID uuid.UUID
}

func (m *mockConsentTokenStore) Get(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error) {
	return nil, nil
}

func (m *mockConsentTokenStore) Store(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, credentialProviderID uuid.UUID, token *UserTokenEntry) error {
	m.key = tenantID.String() + ":" + userID + ":" + serverWorkloadID.String()
	m.credentialProviderID = credentialProviderID
	m.stored = token
	return nil
}

type mockConsentConfigResolver struct {
	config *ConsentConfig
}

func (m *mockConsentConfigResolver) GetConsentConfig(ctx context.Context, tenantID uuid.UUID, serverWorkloadID uuid.UUID, gatewayURL string) (*ConsentConfig, error) {
	return m.config, nil
}

func TestConsentCallback_ExchangesCodeAndStoresToken(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("expected grant_type=authorization_code, got %s", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-auth-code" {
			t.Errorf("expected code=test-auth-code, got %s", r.Form.Get("code"))
		}
		if r.Form.Get("client_id") != "my-client" {
			t.Errorf("expected client_id=my-client, got %s", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "my-secret" {
			t.Errorf("expected client_secret=my-secret, got %s", r.Form.Get("client_secret"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "new-access-token",
			"refresh_token": "new-refresh-token",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	userID := "user@example.com"
	state := tenantID.String() + ":" + userID + ":" + serverWorkloadID.String()

	cpID := uuid.New()
	store := &mockConsentTokenStore{}
	resolver := &mockConsentConfigResolver{
		config: &ConsentConfig{
			TokenEndpoint:        tokenServer.URL,
			ClientID:             "my-client",
			ClientSecret:         "my-secret",
			GatewayCallbackURL:   "https://gateway.example.com/mcp/consent/callback",
			CredentialProviderID: cpID,
		},
	}

	svc := NewConsentCallbackService(store, resolver)

	err := svc.HandleCallback(context.Background(), "test-auth-code", state, TenantScope{TenantID: tenantID, TenantSubdomain: "test", GatewayURL: testGatewayURL})
	if err != nil {
		t.Fatalf("HandleCallback failed: %v", err)
	}

	if store.stored == nil {
		t.Fatal("expected token to be stored")
	}
	if store.stored.AccessToken != "new-access-token" {
		t.Errorf("expected access_token=new-access-token, got %s", store.stored.AccessToken)
	}
	if store.stored.RefreshToken != "new-refresh-token" {
		t.Errorf("expected refresh_token=new-refresh-token, got %s", store.stored.RefreshToken)
	}
	if store.stored.ExpiresAt.Before(time.Now().UTC()) {
		t.Error("expected ExpiresAt in the future")
	}

	expectedKey := tenantID.String() + ":" + userID + ":" + serverWorkloadID.String()
	if store.key != expectedKey {
		t.Errorf("expected store key %s, got %s", expectedKey, store.key)
	}
	if store.credentialProviderID != cpID {
		t.Errorf("expected stored credential provider ID %s, got %s", cpID, store.credentialProviderID)
	}
}

func TestConsentCallback_InvalidState_ReturnsError(t *testing.T) {
	store := &mockConsentTokenStore{}
	resolver := &mockConsentConfigResolver{}
	svc := NewConsentCallbackService(store, resolver)

	err := svc.HandleCallback(context.Background(), "code", "invalid-state", TenantScope{TenantID: uuid.New(), TenantSubdomain: "other", GatewayURL: testGatewayURL})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
}

func TestConsentCallback_TokenEndpointError_ReturnsError(t *testing.T) {
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid_grant"}`))
	}))
	defer tokenServer.Close()

	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	state := tenantID.String() + ":user@test.com:" + serverWorkloadID.String()

	store := &mockConsentTokenStore{}
	resolver := &mockConsentConfigResolver{
		config: &ConsentConfig{
			TokenEndpoint:      tokenServer.URL,
			ClientID:           "client",
			GatewayCallbackURL: "https://gw.example.com/callback",
		},
	}
	svc := NewConsentCallbackService(store, resolver)

	err := svc.HandleCallback(context.Background(), "bad-code", state, TenantScope{TenantID: tenantID, TenantSubdomain: "test", GatewayURL: testGatewayURL})
	if err == nil {
		t.Fatal("expected error for token endpoint failure")
	}
}

func TestParseConsentState_Valid(t *testing.T) {
	tenantID := uuid.New()
	serverID := uuid.New()
	state := tenantID.String() + ":user@test.com:" + serverID.String()

	tid, uid, sid, err := parseConsentState(state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tid != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, tid)
	}
	if uid != "user@test.com" {
		t.Errorf("expected userID user@test.com, got %s", uid)
	}
	if sid != serverID {
		t.Errorf("expected serverWorkloadID %s, got %s", serverID, sid)
	}
}

func TestParseConsentState_InvalidFormat(t *testing.T) {
	_, _, _, err := parseConsentState("only-one-part")
	if err == nil {
		t.Error("expected error for invalid state format")
	}
}

func TestParseConsentState_InvalidTenantUUID(t *testing.T) {
	_, _, _, err := parseConsentState("not-a-uuid:user:" + uuid.New().String())
	if err == nil {
		t.Error("expected error for invalid tenant UUID")
	}
}

func TestParseConsentState_EmptyUserID(t *testing.T) {
	_, _, _, err := parseConsentState(uuid.New().String() + "::" + uuid.New().String())
	if err == nil {
		t.Error("expected error for empty user ID")
	}
}

func TestParseConsentState_InvalidServerUUID(t *testing.T) {
	_, _, _, err := parseConsentState(uuid.New().String() + ":user:not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid server workload UUID")
	}
}

func TestConsentCallback_StateTenantMismatch_Rejected(t *testing.T) {
	store := &mockConsentTokenStore{}
	resolver := &mockConsentConfigResolver{
		config: &ConsentConfig{
			TokenEndpoint:      "https://should-not-be-called.example.com",
			ClientID:           "client",
			GatewayCallbackURL: testGatewayURL + ConsentCallbackPath,
		},
	}
	svc := NewConsentCallbackService(store, resolver)

	stateTenant := uuid.New()
	hostTenant := uuid.New()
	state := stateTenant.String() + ":user@example.com:" + uuid.New().String()

	err := svc.HandleCallback(context.Background(), "code", state,
		TenantScope{TenantID: hostTenant, TenantSubdomain: "host", GatewayURL: testGatewayURL})

	if !errors.Is(err, ErrConsentTenantMismatch) {
		t.Fatalf("expected ErrConsentTenantMismatch, got %v", err)
	}
	if store.stored != nil {
		t.Error("no token may be stored when the tenants disagree")
	}
}

func TestConsentCallback_InvalidScope_Rejected(t *testing.T) {
	store := &mockConsentTokenStore{}
	svc := NewConsentCallbackService(store, &mockConsentConfigResolver{})

	tenantID := uuid.New()
	state := tenantID.String() + ":user@example.com:" + uuid.New().String()

	err := svc.HandleCallback(context.Background(), "code", state, TenantScope{})
	if !errors.Is(err, ErrInvalidTenantScope) {
		t.Fatalf("expected ErrInvalidTenantScope, got %v", err)
	}
	if store.stored != nil {
		t.Error("no token may be stored without a valid tenant scope")
	}
}
