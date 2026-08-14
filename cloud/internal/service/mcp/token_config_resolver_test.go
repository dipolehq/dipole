package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// mockPolicyRepoForResolver implements only what PolicyTokenConfigResolver
// calls: GetByID. Other methods are unused stubs.
type mockPolicyRepoForResolver struct {
	policy *domain.AccessPolicy
	err    error
}

func (m *mockPolicyRepoForResolver) Create(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	return nil
}
func (m *mockPolicyRepoForResolver) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AccessPolicy, error) {
	return m.policy, m.err
}
func (m *mockPolicyRepoForResolver) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	return nil, 0, nil
}
func (m *mockPolicyRepoForResolver) Update(_ context.Context, _ uuid.UUID, _ *domain.AccessPolicy) error {
	return nil
}
func (m *mockPolicyRepoForResolver) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *mockPolicyRepoForResolver) FindByServerTarget(_ context.Context, _ uuid.UUID, _ string, _ int, _ *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}
func (m *mockPolicyRepoForResolver) FindByAppProtocol(_ context.Context, _ uuid.UUID, _ string) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}
func (m *mockPolicyRepoForResolver) ListActiveWithRelations(_ context.Context, _ uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	return nil, nil
}

type mockCPRepoForResolver struct {
	cp  *domain.CredentialProvider
	err error
}

func (m *mockCPRepoForResolver) Create(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	return nil
}
func (m *mockCPRepoForResolver) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.CredentialProvider, error) {
	return m.cp, m.err
}
func (m *mockCPRepoForResolver) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	return nil, 0, nil
}
func (m *mockCPRepoForResolver) Update(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	return nil
}
func (m *mockCPRepoForResolver) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

type mockCWRepoForResolver struct {
	cw  *domain.ClientWorkload
	err error
}

func (m *mockCWRepoForResolver) Create(_ context.Context, _ uuid.UUID, _ *domain.ClientWorkload) error {
	return nil
}
func (m *mockCWRepoForResolver) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.ClientWorkload, error) {
	return m.cw, m.err
}
func (m *mockCWRepoForResolver) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.ClientWorkload, int, error) {
	return nil, 0, nil
}
func (m *mockCWRepoForResolver) Update(_ context.Context, _ uuid.UUID, _ *domain.ClientWorkload) error {
	return nil
}
func (m *mockCWRepoForResolver) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func oidcCPConfigJSON(t *testing.T) []byte {
	t.Helper()
	cfg := oidcCPConfig{
		Audience:        "https://mcp.example.com",
		Subject:         "user@example.com",
		LifetimeSeconds: 3600,
		Algorithm:       "ES256",
	}
	b, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// The Client Workload resolved by the /authorize redirect_uri match must be
// carried into the TokenConfig, so it can be embedded in the issued JWT and
// later read back by the Gateway's left-side auth. No second matching logic —
// this test can only pass by reusing the ClientWorkloadID already on the
// policy loaded for Credential Provider resolution.
func TestResolveTokenConfig_PopulatesClientWorkloadFromPolicy(t *testing.T) {
	tenantID := uuid.New()
	policyID := uuid.New()
	cpID := uuid.New()
	cwID := uuid.New()

	policyRepo := &mockPolicyRepoForResolver{
		policy: &domain.AccessPolicy{
			ID:               policyID,
			TenantID:         tenantID,
			ClientWorkloadID: cwID,
			CredentialProviderIDs: []domain.PolicyCredentialProvider{
				{CredentialProviderID: cpID},
			},
		},
	}
	cpRepo := &mockCPRepoForResolver{
		cp: &domain.CredentialProvider{
			ID:              cpID,
			Type:            domain.CredentialProviderTypeOIDCIDToken,
			ConfigEncrypted: oidcCPConfigJSON(t),
		},
	}
	cwRepo := &mockCWRepoForResolver{
		cw: &domain.ClientWorkload{
			ID:   cwID,
			Name: "Claude Desktop",
		},
	}

	resolver := NewPolicyTokenConfigResolver(policyRepo, cpRepo, cwRepo, nil)

	cfg, err := resolver.ResolveTokenConfig(policyID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ClientWorkloadName != "Claude Desktop" {
		t.Errorf("expected ClientWorkloadName=%q, got %q", "Claude Desktop", cfg.ClientWorkloadName)
	}
	if cfg.ClientWorkloadID != cwID {
		t.Errorf("expected ClientWorkloadID=%s, got %s", cwID, cfg.ClientWorkloadID)
	}
}

// A Client Workload lookup failure (e.g. deleted after the policy referenced
// it) must never block token issuance — the security decision already
// happened when the policy matched at /authorize time. The resolver must
// degrade to an empty name, not an error.
func TestResolveTokenConfig_ClientWorkloadLookupFails_StillSucceeds(t *testing.T) {
	tenantID := uuid.New()
	policyID := uuid.New()
	cpID := uuid.New()
	cwID := uuid.New()

	policyRepo := &mockPolicyRepoForResolver{
		policy: &domain.AccessPolicy{
			ID:               policyID,
			TenantID:         tenantID,
			ClientWorkloadID: cwID,
			CredentialProviderIDs: []domain.PolicyCredentialProvider{
				{CredentialProviderID: cpID},
			},
		},
	}
	cpRepo := &mockCPRepoForResolver{
		cp: &domain.CredentialProvider{
			ID:              cpID,
			Type:            domain.CredentialProviderTypeOIDCIDToken,
			ConfigEncrypted: oidcCPConfigJSON(t),
		},
	}
	cwRepo := &mockCWRepoForResolver{err: domain.ErrNotFound}

	resolver := NewPolicyTokenConfigResolver(policyRepo, cpRepo, cwRepo, nil)

	cfg, err := resolver.ResolveTokenConfig(policyID, tenantID)
	if err != nil {
		t.Fatalf("expected token config resolution to succeed despite CW lookup failure, got: %v", err)
	}
	if cfg.ClientWorkloadName != "" {
		t.Errorf("expected empty ClientWorkloadName on lookup failure, got %q", cfg.ClientWorkloadName)
	}
	if cfg.Audience != "https://mcp.example.com" {
		t.Errorf("expected credential provider config still resolved correctly, got audience=%q", cfg.Audience)
	}
}

// A nil cwRepo (e.g. not wired by some future caller) must also degrade
// gracefully rather than panic.
func TestResolveTokenConfig_NilClientWorkloadRepo_StillSucceeds(t *testing.T) {
	tenantID := uuid.New()
	policyID := uuid.New()
	cpID := uuid.New()

	policyRepo := &mockPolicyRepoForResolver{
		policy: &domain.AccessPolicy{
			ID:       policyID,
			TenantID: tenantID,
			CredentialProviderIDs: []domain.PolicyCredentialProvider{
				{CredentialProviderID: cpID},
			},
		},
	}
	cpRepo := &mockCPRepoForResolver{
		cp: &domain.CredentialProvider{
			ID:              cpID,
			Type:            domain.CredentialProviderTypeOIDCIDToken,
			ConfigEncrypted: oidcCPConfigJSON(t),
		},
	}

	resolver := NewPolicyTokenConfigResolver(policyRepo, cpRepo, nil, nil)

	cfg, err := resolver.ResolveTokenConfig(policyID, tenantID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ClientWorkloadName != "" {
		t.Errorf("expected empty ClientWorkloadName with nil cwRepo, got %q", cfg.ClientWorkloadName)
	}
}

func (m *mockPolicyRepoForResolver) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }
func (m *mockCPRepoForResolver) CountInactive(_ context.Context, _ uuid.UUID) int     { return 0 }
func (m *mockCWRepoForResolver) CountInactive(_ context.Context, _ uuid.UUID) int     { return 0 }

func (m *mockPolicyRepoForResolver) IsHostnameAuthorizedForTLS(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return false, nil
}
