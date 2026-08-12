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

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil)

	// Alice gets Alice's token
	cred, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "alice-jira-token" {
		t.Errorf("expected alice-jira-token, got %s", cred.AccessToken)
	}

	// Bob gets Bob's token
	cred, err = rightAuth.GetCredential(context.Background(), "bob@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "bob-jira-token" {
		t.Errorf("expected bob-jira-token, got %s", cred.AccessToken)
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

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)

	_, err := rightAuth.GetCredential(context.Background(), "newuser@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != ErrConsentRequired {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}
}

func TestRightAuth_NoPolicyForServer_Denied(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{policies: nil}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)

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

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil)

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

func (r *mockTokenRefresher) Refresh(_ context.Context, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error) {
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

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil)

	cred, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", testScope(tenantID), "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "new-access-token" {
		t.Errorf("expected new-access-token, got %s", cred.AccessToken)
	}
	if refresher.lastToken != "refresh-token-123" {
		t.Errorf("expected refresh token 'refresh-token-123', got '%s'", refresher.lastToken)
	}
	if refresher.lastServerID != serverID {
		t.Errorf("expected server ID %s, got %s", serverID, refresher.lastServerID)
	}

	// Verify the new token was stored
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

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil)

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
		RefreshToken: "", // No refresh token
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

	rightAuth := NewRightSideAuth(policyFinder, store, refresher, nil)

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

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil)

	creds, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", testScope(tenantID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 2 {
		t.Fatalf("expected 2 credentials, got %d", len(creds))
	}

	credMap := make(map[string]string)
	for _, c := range creds {
		credMap[c.ServerWorkload.Name] = c.AccessToken
	}
	if credMap["jira"] != "alice-jira-token" {
		t.Errorf("expected jira token 'alice-jira-token', got '%s'", credMap["jira"])
	}
	if credMap["slack"] != "alice-slack-token" {
		t.Errorf("expected slack token 'alice-slack-token', got '%s'", credMap["slack"])
	}
}

func TestRightAuth_GetAllServerCredentials_SkipsUnavailable(t *testing.T) {
	jiraID := uuid.New()
	slackID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	// Only jira has a stored token, slack does not
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

	rightAuth := NewRightSideAuth(policyFinder, store, nil, nil)

	creds, err := rightAuth.GetAllServerCredentials(context.Background(), "alice@corp.com", testScope(tenantID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should return only jira (slack skipped because no token → ErrConsentRequired)
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
			{ServerWorkload: ServerInfo{ID: jiraID, Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-shared"},
			{ServerWorkload: ServerInfo{ID: slackID, Name: "slack"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "slack-shared"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)

	cred, err := rightAuth.GetCredentialByServerName(context.Background(), "alice@corp.com", testScope(tenantID), toolPrefix(ServerInfo{ID: slackID, Name: "slack"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "slack-shared" {
		t.Errorf("expected slack-shared, got '%s'", cred.AccessToken)
	}
	if cred.ServerWorkload.Name != "slack" {
		t.Errorf("expected server name 'slack', got '%s'", cred.ServerWorkload.Name)
	}
}

func TestRightAuth_GetCredentialByServerName_NotFound(t *testing.T) {
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{ServerWorkload: ServerInfo{ID: uuid.New(), Name: "jira"}, CredentialType: "oauth_authorization_code", SharedAccessToken: "jira-token"},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), nil, nil)

	_, err := rightAuth.GetCredentialByServerName(context.Background(), "alice@corp.com", testScope(tenantID), "unknown-server")
	if err != ErrNoPolicyForServer {
		t.Fatalf("expected ErrNoPolicyForServer, got %v", err)
	}
}

// An unattributed request must not reach the policy finder at all. Without this
// guard a zero TenantScope would query policies for the nil tenant and match a
// Client Workload whose audience identifier happened to be empty.
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
			rightAuth := NewRightSideAuth(finder, newMockTokenStore(), nil, nil)

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

// The scope's tenant and Gateway URL must both reach the policy finder — those
// two values are what scope a lookup to one tenant's policies.
func TestRightAuth_PassesScopeThroughToPolicyFinder(t *testing.T) {
	tenantID := uuid.New()
	finder := &countingPolicyFinder{}
	rightAuth := NewRightSideAuth(finder, newMockTokenStore(), nil, nil)
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
