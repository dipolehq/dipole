// identity_providers.go implements business logic and validation for Identity Providers.
package resources

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

var validIdentityProviderTypes = map[string]struct{}{
	"saml_2.0": {},
	"oidc_1.0": {},
}

type IdentityProviderService struct {
	repo repository.IdentityProviderRepository
}

func NewIdentityProviderService(repo repository.IdentityProviderRepository) *IdentityProviderService {
	return &IdentityProviderService{repo: repo}
}

func (s *IdentityProviderService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.IdentityProvider, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *IdentityProviderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.IdentityProvider, error) {
	idp, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if idp == nil {
		return nil, domain.ErrNotFound
	}
	return idp, nil
}

func (s *IdentityProviderService) Create(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error {
	normalizeIdentityProvider(idp)
	if err := validateIdentityProvider(idp); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, idp)
}

func (s *IdentityProviderService) Update(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error {
	normalizeIdentityProvider(idp)
	if err := validateIdentityProvider(idp); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, idp.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	idp.Config = preserveSecrets(existing.Config, idp.Config)
	return s.repo.Update(ctx, tenantID, idp)
}

func (s *IdentityProviderService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeIdentityProvider(idp *domain.IdentityProvider) {
	idp.Name = strings.TrimSpace(idp.Name)
	idp.Description = strings.TrimSpace(idp.Description)
	idp.Type = strings.TrimSpace(idp.Type)
}

func validateIdentityProvider(idp *domain.IdentityProvider) error {
	if idp.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if idp.Type == "" {
		return domain.NewValidationError("type", "type is required")
	}
	if _, ok := validIdentityProviderTypes[idp.Type]; !ok {
		return domain.NewValidationError("type", "type must be saml_2.0 or oidc_1.0")
	}
	return nil
}

func preserveSecrets(existingRaw, incomingRaw json.RawMessage) json.RawMessage {
	secretFields := []string{"clientSecret"}

	var existing, incoming map[string]json.RawMessage
	if json.Unmarshal(existingRaw, &existing) != nil {
		return incomingRaw
	}
	if json.Unmarshal(incomingRaw, &incoming) != nil {
		return incomingRaw
	}

	for _, field := range secretFields {
		inVal, inOK := incoming[field]
		exVal, exOK := existing[field]
		if !exOK {
			continue
		}
		if !inOK || string(inVal) == `""` || string(inVal) == `null` || string(inVal) == "" {
			incoming[field] = exVal
		}
	}

	merged, err := json.Marshal(incoming)
	if err != nil {
		return incomingRaw
	}
	return merged
}
