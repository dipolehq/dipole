/*
user_consent_test.go tests the UserConsentService that builds OAuth consent URLs
for per-user MCP server authorization.
*/
package mcp_gateway

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type mockConsentResolver struct {
	config *ConsentConfig
}

func (m *mockConsentResolver) GetConsentConfig(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (*ConsentConfig, error) {
	return m.config, nil
}

func TestBuildConsentURL_CorrectFormat(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	userID := "alice@corp.com"

	resolver := &mockConsentResolver{
		config: &ConsentConfig{
			AuthorizationEndpoint: "https://auth.jira.com/authorize",
			TokenEndpoint:         "https://auth.jira.com/token",
			ClientID:              "jira-client",
			ClientSecret:          "jira-secret",
			Scopes:                "read:jira write:jira",
			GatewayCallbackURL:    "https://gw.example.com/mcp/consent/callback",
		},
	}

	svc := NewUserConsentService(newMockTokenStore(), resolver)

	result, err := svc.BuildConsentURL(context.Background(), TenantScope{TenantID: tenantID, GatewayURL: testGatewayURL}, userID, serverWorkloadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil || result.ConsentURL == "" {
		t.Fatal("expected non-empty consent URL")
	}

	// Parse the URL and verify query params
	parsed, err := url.Parse(result.ConsentURL)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}

	if parsed.Scheme != "https" || parsed.Host != "auth.jira.com" || parsed.Path != "/authorize" {
		t.Errorf("unexpected base URL: %s", result.ConsentURL)
	}

	q := parsed.Query()
	if q.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got '%s'", q.Get("response_type"))
	}
	if q.Get("client_id") != "jira-client" {
		t.Errorf("expected client_id=jira-client, got '%s'", q.Get("client_id"))
	}
	if q.Get("redirect_uri") != "https://gw.example.com/mcp/consent/callback" {
		t.Errorf("expected redirect_uri to match callback URL, got '%s'", q.Get("redirect_uri"))
	}
	if q.Get("scope") != "read:jira write:jira" {
		t.Errorf("expected scope='read:jira write:jira', got '%s'", q.Get("scope"))
	}

	// Verify state contains tenant:user:server
	state := q.Get("state")
	if !strings.Contains(state, tenantID.String()) {
		t.Error("state missing tenant ID")
	}
	if !strings.Contains(state, userID) {
		t.Error("state missing user ID")
	}
	if !strings.Contains(state, serverWorkloadID.String()) {
		t.Error("state missing server workload ID")
	}
}

func TestBuildConsentURL_StateFormat(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	userID := "bob@test.com"

	resolver := &mockConsentResolver{
		config: &ConsentConfig{
			AuthorizationEndpoint: "https://auth.example.com/authorize",
			ClientID:              "client",
			GatewayCallbackURL:    "https://gw.example.com/callback",
		},
	}

	svc := NewUserConsentService(newMockTokenStore(), resolver)

	result, err := svc.BuildConsentURL(context.Background(), TenantScope{TenantID: tenantID, GatewayURL: testGatewayURL}, userID, serverWorkloadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Extract state from URL
	parsed, _ := url.Parse(result.ConsentURL)
	state := parsed.Query().Get("state")

	// State should be exactly: tenantID:userID:serverWorkloadID
	expected := tenantID.String() + ":" + userID + ":" + serverWorkloadID.String()
	if state != expected {
		t.Errorf("expected state '%s', got '%s'", expected, state)
	}
}
