package mcp_gateway

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockAccessPolicyRepo struct {
	byProtocol    []domain.AccessPolicyWithRelations
	byTarget      []domain.AccessPolicyWithRelations
	byProtocolErr error
	byTargetErr   error
}

func (m *mockAccessPolicyRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	panic("not used")
}
func (m *mockAccessPolicyRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AccessPolicy, error) {
	panic("not used")
}
func (m *mockAccessPolicyRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	panic("not used")
}
func (m *mockAccessPolicyRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	panic("not used")
}
func (m *mockAccessPolicyRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	panic("not used")
}

func (m *mockAccessPolicyRepo) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int, _ *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.byTarget, m.byTargetErr
}

func (m *mockAccessPolicyRepo) FindByAppProtocol(_ context.Context, _ uuid.UUID, _ string) ([]domain.AccessPolicyWithRelations, error) {
	return m.byProtocol, m.byProtocolErr
}

func (m *mockAccessPolicyRepo) ListActiveWithRelations(_ context.Context, _ uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return m.byProtocol, m.byProtocolErr
}

func buildMCPPolicy(gatewayAudience string, serverName string, serverHost string, serverPort int) domain.AccessPolicyWithRelations {
	serverID := uuid.New()
	cpID := uuid.New()

	return domain.AccessPolicyWithRelations{
		Policy: domain.AccessPolicy{
			ID:       uuid.New(),
			IsActive: true,
			CredentialProviderIDs: []domain.PolicyCredentialProvider{
				{CredentialProviderID: cpID},
			},
		},
		ClientWorkload: domain.ClientWorkload{
			ID:   uuid.New(),
			Name: "Gateway Client",
			Identifiers: []domain.Identifier{
				{Type: domain.IdentifierTypeOIDCIDTokenAudience, Value: gatewayAudience},
			},
		},
		ServerWorkload: domain.ServerWorkload{
			ID:          serverID,
			Name:        serverName,
			Host:        serverHost,
			Port:        serverPort,
			AppProtocol: domain.AppProtocolMCP,
		},
		CredentialProviders: []domain.CredentialProvider{
			{ID: cpID, Type: domain.CredentialProviderTypeMCPUserBased},
		},
	}
}

func TestDBPolicyFinder_FindGatewayPolicies_MatchesAudience(t *testing.T) {
	policy := buildMCPPolicy("https://gw.example.com/", "jira", "jira.example.com", 443)

	repo := &mockAccessPolicyRepo{byProtocol: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	results, err := finder.FindGatewayPolicies(context.Background(), uuid.New(), "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(results))
	}
	if results[0].ServerWorkload.Name != "jira" {
		t.Errorf("expected server name 'jira', got '%s'", results[0].ServerWorkload.Name)
	}
}

func TestDBPolicyFinder_FindGatewayPolicies_TrailingSlashNormalized(t *testing.T) {
	policy := buildMCPPolicy("https://gw.example.com", "jira", "jira.example.com", 443)

	repo := &mockAccessPolicyRepo{byProtocol: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	results, err := finder.FindGatewayPolicies(context.Background(), uuid.New(), "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 policy after trailing slash normalization, got %d", len(results))
	}
}

func TestDBPolicyFinder_FindGatewayPolicies_StoredIdentifierHasTrailingSlash(t *testing.T) {
	policy := buildMCPPolicy("https://gw.example.com/", "jira", "jira.example.com", 443)

	repo := &mockAccessPolicyRepo{byProtocol: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	results, err := finder.FindGatewayPolicies(context.Background(), uuid.New(), "https://gw.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 policy when the stored identifier carries the slash, got %d", len(results))
	}
}

func TestDBPolicyFinder_FindGatewayPolicies_SkipsNonMatchingAudience(t *testing.T) {
	policy := buildMCPPolicy("https://other.example.com/", "github", "github.example.com", 443)

	repo := &mockAccessPolicyRepo{byProtocol: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	results, err := finder.FindGatewayPolicies(context.Background(), uuid.New(), "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 policies for non-matching audience, got %d", len(results))
	}
}

func TestDBPolicyFinder_FindGatewayPolicies_EmptyResult(t *testing.T) {
	repo := &mockAccessPolicyRepo{byProtocol: []domain.AccessPolicyWithRelations{}}
	finder := NewDBRightSidePolicyFinder(repo)

	results, err := finder.FindGatewayPolicies(context.Background(), uuid.New(), "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 policies, got %d", len(results))
	}
}

func TestDBPolicyFinder_FindPolicyForServer_ReturnsMatch(t *testing.T) {
	policy := buildMCPPolicy("https://gw.example.com/", "jira", "jira.example.com", 443)

	repo := &mockAccessPolicyRepo{byTarget: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	result, err := finder.FindPolicyForServer(context.Background(), uuid.New(), "https://gw.example.com/", "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.ServerWorkload.Name != "jira" {
		t.Errorf("expected server name 'jira', got '%s'", result.ServerWorkload.Name)
	}
	if result.CredentialType != domain.CredentialProviderTypeMCPUserBased {
		t.Errorf("expected credential type '%s', got '%s'", domain.CredentialProviderTypeMCPUserBased, result.CredentialType)
	}
}

func TestDBPolicyFinder_FindPolicyForServer_NoMatch_ReturnsNil(t *testing.T) {
	repo := &mockAccessPolicyRepo{byTarget: []domain.AccessPolicyWithRelations{}}
	finder := NewDBRightSidePolicyFinder(repo)

	result, err := finder.FindPolicyForServer(context.Background(), uuid.New(), "https://gw.example.com/", "unknown.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result for non-matching server")
	}
}

func TestDBPolicyFinder_FindPolicyForServer_SkipsNonMCPProtocol(t *testing.T) {
	policy := buildMCPPolicy("https://gw.example.com/", "http-svc", "http.example.com", 443)
	policy.ServerWorkload.AppProtocol = domain.AppProtocolHTTP

	repo := &mockAccessPolicyRepo{byTarget: []domain.AccessPolicyWithRelations{policy}}
	finder := NewDBRightSidePolicyFinder(repo)

	result, err := finder.FindPolicyForServer(context.Background(), uuid.New(), "https://gw.example.com/", "http.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result for non-MCP protocol policy")
	}
}

func (m *mockAccessPolicyRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }

func (m *mockAccessPolicyRepo) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}
