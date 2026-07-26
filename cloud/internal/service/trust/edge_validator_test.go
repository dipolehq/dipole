/*
edge_validator_test.go tests the EdgeTrustValidatorService that loads Trust
Providers by ID and delegates to the real trust validation service.
*/
package trust

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type mockTrustProviderRepo struct {
	tp  *domain.TrustProvider
	err error
}

func (m *mockTrustProviderRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	panic("not used")
}
func (m *mockTrustProviderRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.TrustProvider, error) {
	return m.tp, m.err
}
func (m *mockTrustProviderRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.TrustProvider, int, error) {
	panic("not used")
}
func (m *mockTrustProviderRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.TrustProvider) error {
	panic("not used")
}
func (m *mockTrustProviderRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	panic("not used")
}

type mockTrustValidationService struct {
	result *domain.TrustProviderResult
	err    error
}

func (m *mockTrustValidationService) Validate(_ context.Context, _ *domain.TrustProvider, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return m.result, m.err
}

func TestEdgeTrustValidator_ValidTP_Attested(t *testing.T) {
	tenantID := uuid.New()
	tpID := uuid.New()

	repo := &mockTrustProviderRepo{
		tp: &domain.TrustProvider{
			ID:       tpID,
			TenantID: tenantID,
			IsActive: true,
			Type:     "github_action",
			Name:     "GitHub Production",
		},
	}

	svc := NewTrustValidationService(NewJWKSCache(nil), nil)
	svc.RegisterValidator("github_action", &stubValidator{
		result: &domain.TrustProviderResult{ProviderID: tpID, Attested: true},
	})

	edgeValidator := NewEdgeTrustValidator(repo, svc)

	result, err := edgeValidator.ValidateByID(tenantID, tpID, &domain.ClientEvidence{
		GitHub: &domain.IdentityTokenEvidence{IdentityToken: "test-jwt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Error("expected attested=true")
	}
}

func TestEdgeTrustValidator_TPNotFound_ReturnsDeny(t *testing.T) {
	repo := &mockTrustProviderRepo{tp: nil, err: nil}
	svc := NewTrustValidationService(NewJWKSCache(nil), nil)
	edgeValidator := NewEdgeTrustValidator(repo, svc)

	result, err := edgeValidator.ValidateByID(uuid.New(), uuid.New(), &domain.ClientEvidence{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for not-found TP")
	}
	if result.Reason != "TrustProviderNotFound" {
		t.Errorf("expected reason 'TrustProviderNotFound', got '%s'", result.Reason)
	}
}

func TestEdgeTrustValidator_TPInactive_ReturnsDeny(t *testing.T) {
	repo := &mockTrustProviderRepo{
		tp: &domain.TrustProvider{
			ID:       uuid.New(),
			IsActive: false,
			Name:     "Disabled TP",
		},
	}
	svc := NewTrustValidationService(NewJWKSCache(nil), nil)
	edgeValidator := NewEdgeTrustValidator(repo, svc)

	result, err := edgeValidator.ValidateByID(uuid.New(), uuid.New(), &domain.ClientEvidence{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for inactive TP")
	}
	if result.Reason != "TrustProviderInactive" {
		t.Errorf("expected reason 'TrustProviderInactive', got '%s'", result.Reason)
	}
}

func TestEdgeTrustValidator_DBError_ReturnsDeny(t *testing.T) {
	repo := &mockTrustProviderRepo{tp: nil, err: fmt.Errorf("db connection failed")}
	svc := NewTrustValidationService(NewJWKSCache(nil), nil)
	edgeValidator := NewEdgeTrustValidator(repo, svc)

	result, err := edgeValidator.ValidateByID(uuid.New(), uuid.New(), &domain.ClientEvidence{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false on DB error")
	}
	if result.Reason != "TrustProviderNotFound" {
		t.Errorf("expected reason 'TrustProviderNotFound', got '%s'", result.Reason)
	}
}

func TestEdgeTrustValidator_ValidationFails_ReturnsNonAttested(t *testing.T) {
	tpID := uuid.New()
	repo := &mockTrustProviderRepo{
		tp: &domain.TrustProvider{
			ID:       tpID,
			IsActive: true,
			Type:     "github_action",
			Name:     "GitHub",
		},
	}

	svc := NewTrustValidationService(NewJWKSCache(nil), nil)
	svc.RegisterValidator("github_action", &stubValidator{
		result: &domain.TrustProviderResult{ProviderID: tpID, Attested: false, Reason: "InvalidSignature"},
	})

	edgeValidator := NewEdgeTrustValidator(repo, svc)

	result, err := edgeValidator.ValidateByID(uuid.New(), tpID, &domain.ClientEvidence{
		GitHub: &domain.IdentityTokenEvidence{IdentityToken: "bad-jwt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for invalid signature")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

// stubValidator implements TypeValidator for testing
type stubValidator struct {
	result *domain.TrustProviderResult
	err    error
}

func (v *stubValidator) Validate(_ context.Context, _ *domain.TrustProvider, _ *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	return v.result, v.err
}
