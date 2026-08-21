package mcp_gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRightAuth_PerUserIsolation(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "alice-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}
	store.tokens["bob@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "bob-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil, nil)

	cred, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Authorization != "Bearer alice-jira-token" {
		t.Errorf("expected 'Bearer alice-jira-token', got %s", cred.Authorization)
	}

	cred, err = rightAuth.GetCredential(context.Background(), "bob@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Authorization != "Bearer bob-jira-token" {
		t.Errorf("expected 'Bearer bob-jira-token', got %s", cred.Authorization)
	}
}

func TestRightAuth_FirstTimeAccess_ConsentRequired(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "newuser@corp.com", testScope(tenantID), "jira.example.com", 443)
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}
	var consentErr *ConsentRequiredError
	if !errors.As(err, &consentErr) {
		t.Fatalf("expected *ConsentRequiredError, got %T", err)
	}
	if consentErr.ServerWorkloadID != serverID {
		t.Errorf("expected ServerWorkloadID %s, got %s", serverID, consentErr.ServerWorkloadID)
	}
}

func TestRightAuth_NoPolicyForServer_Denied(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{policies: nil}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "unknown.example.com", 443)
	if err != ErrNoPolicyForServer {
		t.Fatalf("expected ErrNoPolicyForServer, got %v", err)
	}
}

func TestRightAuth_ExpiredToken_Error(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrCredentialExpired {
		t.Fatalf("expected ErrCredentialExpired, got %v", err)
	}
}

type mockTokenRefresher struct {
	refreshedEntry *UserTokenEntry
	err            error
	lastToken      string
	lastServerID   uuid.UUID
}

func (r *mockTokenRefresher) Refresh(_ context.Context, _ TenantScope, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error) {
	r.lastToken = refreshToken
	r.lastServerID = serverWorkloadID
	if r.err != nil {
		return nil, r.err
	}
	return r.refreshedEntry, nil
}

func TestRightAuth_ExpiredToken_RefreshSuccess(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token-123",
		ExpiresAt:    time.Now().UTC().Add(-1 * time.Hour),
	}

	refresher := &mockTokenRefresher{
		refreshedEntry: &UserTokenEntry{
			AccessToken:  "new-access-token",
			RefreshToken: "new-refresh-token",
			ExpiresAt:    time.Now().UTC().Add(1 * time.Hour),
		},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil, nil)

	cred, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Authorization != "Bearer new-access-token" {
		t.Errorf("expected 'Bearer new-access-token', got %s", cred.Authorization)
	}
	if refresher.lastToken != "refresh-token-123" {
		t.Errorf("expected refresh token 'refresh-token-123', got '%s'", refresher.lastToken)
	}
	if refresher.lastServerID != serverID {
		t.Errorf("expected server ID %s, got %s", serverID, refresher.lastServerID)
	}

	stored := store.tokens["alice@corp.com:"+serverID.String()]
	if stored.AccessToken != "new-access-token" {
		t.Errorf("expected stored token 'new-access-token', got '%s'", stored.AccessToken)
	}
	if stored.RefreshToken != "new-refresh-token" {
		t.Errorf("expected stored refresh token 'new-refresh-token', got '%s'", stored.RefreshToken)
	}
}

func TestRightAuth_ExpiredToken_RefreshFails_ReturnsExpired(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken:  "expired-token",
		RefreshToken: "refresh-token-123",
		ExpiresAt:    time.Now().UTC().Add(-1 * time.Hour),
	}

	refresher := &mockTokenRefresher{
		err: errors.New("refresh token revoked"),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrCredentialExpired {
		t.Fatalf("expected ErrCredentialExpired, got %v", err)
	}
}

func TestRightAuth_ExpiredToken_NoRefreshToken_ReturnsExpired(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken:  "expired-token",
		RefreshToken: "",
		ExpiresAt:    time.Now().UTC().Add(-1 * time.Hour),
	}

	refresher := &mockTokenRefresher{
		refreshedEntry: &UserTokenEntry{AccessToken: "should-not-reach"},
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrCredentialExpired {
		t.Fatalf("expected ErrCredentialExpired, got %v", err)
	}
}

func TestRightAuth_GetAllServerCredentials_ReturnsAllValid(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+jiraID.String()] = &UserTokenEntry{
		AccessToken: "alice-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}
	store.tokens["alice@corp.com:"+slackID.String()] = &UserTokenEntry{
		AccessToken: "alice-slack-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira", Host: "jira.example.com", Port: 443}, CredentialType: "mcp_user_based_access_token"},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack", Host: "slack.example.com", Port: 443}, CredentialType: "mcp_user_based_access_token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil, nil)

	creds, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", testScope(tenantID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}

	credMap := make(map[string]string)
	for _, c := range creds {
		credMap[c.ServerWorkload.Name] = c.Authorization
	}
	if credMap["jira"] != "Bearer alice-jira-token" {
		t.Errorf("expected jira token 'Bearer alice-jira-token', got '%s'", credMap["jira"])
	}
	if credMap["slack"] != "Bearer alice-slack-token" {
		t.Errorf("expected slack token 'Bearer alice-slack-token', got '%s'", credMap["slack"])
	}
}

func TestRightAuth_GetAllServerCredentials_SkipsUnavailable(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+jiraID.String()] = &UserTokenEntry{
		AccessToken: "alice-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "mcp_user_based_access_token"},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack"}, CredentialType: "mcp_user_based_access_token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil, nil)

	creds, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", testScope(tenantID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 1 {
		t.Fatalf("expected 1 credential (slack skipped), got %d", len(creds))
	}
	if creds[0].ServerWorkload.Name != "jira" {
		t.Errorf("expected jira, got '%s'", creds[0].ServerWorkload.Name)
	}
}

func TestRightAuth_GetCredentialByServerName_Found(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", CredentialConfigEncrypted: []byte("jira-shared")},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack"}, CredentialType: "oauth_authorization_code", CredentialConfigEncrypted: []byte("slack-shared")},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, &mockSharedOAuthResolver{})

	cred, err := rightAuth.GetCredentialByServerName(context.Background(), "alice@corp.com", testScope(tenantID), toolPrefix(ServerInfo{ID: slackID, Name: "slack"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.Authorization != "Bearer slack-shared" {
		t.Errorf("expected 'Bearer slack-shared', got '%s'", cred.Authorization)
	}
	if cred.ServerWorkload.Name != "slack" {
		t.Errorf("expected server name 'slack', got '%s'", cred.ServerWorkload.Name)
	}
}

func TestRightAuth_GetCredentialByServerName_NotFound(t *testing.T) {
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: uuid.New(), Name: "jira"}, CredentialType: "oauth_authorization_code", CredentialConfigEncrypted: []byte("jira-token")},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, &mockSharedOAuthResolver{})

	_, err := rightAuth.GetCredentialByServerName(context.Background(), "alice@corp.com", testScope(tenantID), "unknown-server")
	if err != ErrNoPolicyForServer {
		t.Fatalf("expected ErrNoPolicyForServer, got %v", err)
	}
}

func TestRightAuth_SharedOAuth_NoResolverConfigured_ReturnsConfigError(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443}, CredentialType: "oauth_authorization_code"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrCredentialProviderNotConfigured {
		t.Fatalf("expected ErrCredentialProviderNotConfigured, got %v", err)
	}
}

func TestRightAuth_APIKeyDecrypter_NotConfigured_ReturnsConfigError(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443}, CredentialType: "api_key"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrCredentialProviderNotConfigured {
		t.Fatalf("expected ErrCredentialProviderNotConfigured, got %v", err)
	}
}

func TestRightAuth_InvalidTenantScope_FailsClosed(t *testing.T) {
	scopes := map[string]TenantScope{
		"zero value":         {},
		"missing tenant":     {TenantSubdomain: "test", GatewayURL: testGatewayURL},
		"missing gatewayURL": {TenantID: uuid.New(), TenantSubdomain: "test"},
		"missing subdomain":  {TenantID: uuid.New(), GatewayURL: testGatewayURL},
	}

	for name, scope := range scopes {
		t.Run(name, func(t *testing.T) {
			finder := &countingPolicyFinder{}
			rightAuth := NewRightSideAuth(finder, newMockTokenStore(), nil, nil, nil)

			if _, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", scope, "jira.example.com", 443); err != ErrInvalidTenantScope {
				t.Errorf("GetCredential: expected ErrInvalidTenantScope, got %v", err)
			}
			if _, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", scope); err != ErrInvalidTenantScope {
				t.Errorf("GetAllServerCredentials: expected ErrInvalidTenantScope, got %v", err)
			}
			if _, err := rightAuth.GetCredentialByServerName(context.Background(), "alice@corp.com", scope, "jira"); err != ErrInvalidTenantScope {
				t.Errorf("GetCredentialByServerName: expected ErrInvalidTenantScope, got %v", err)
			}
			if finder.calls != 0 {
				t.Errorf("policy finder must not be consulted, got %d calls", finder.calls)
			}
		})
	}
}

func TestRightAuth_PassesScopeThroughToPolicyFinder(t *testing.T) {
	tenantID := uuid.New()
	finder := &countingPolicyFinder{}
	rightAuth := NewRightSideAuth(finder, newMockTokenStore(), nil, nil, nil)
	scope := TenantScope{TenantID: tenantID, TenantSubdomain: "test", GatewayURL: testGatewayURL}

	if _, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", scope); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if finder.gotTenantID != tenantID {
		t.Errorf("expected tenant %s, got %s", tenantID, finder.gotTenantID)
	}
	if finder.gotGatewayURL != testGatewayURL {
		t.Errorf("expected gateway URL %q, got %q", testGatewayURL, finder.gotGatewayURL)
	}
}

type countingPolicyFinder struct {
	calls         int
	gotTenantID   uuid.UUID
	gotGatewayURL string
}

func (f *countingPolicyFinder) FindGatewayPolicies(_ context.Context, tenantID uuid.UUID, gatewayURL string) ([]RightSidePolicy, error) {
	f.calls++
	f.gotTenantID = tenantID
	f.gotGatewayURL = gatewayURL
	return nil, nil
}

func (f *countingPolicyFinder) FindPolicyForServer(_ context.Context, tenantID uuid.UUID, gatewayURL string, _ string, _ int) (*RightSidePolicy, error) {
	f.calls++
	f.gotTenantID = tenantID
	f.gotGatewayURL = gatewayURL
	return nil, nil
}
