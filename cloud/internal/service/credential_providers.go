/*
Package service — credential_providers.go implements business logic for
Credential Providers including encryption/decryption of config at rest.
*/
package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type CredentialProviderService struct {
	repo          repository.CredentialProviderRepository
	encryptionKey []byte
	limiter       *PlanLimiter
}

func NewCredentialProviderService(repo repository.CredentialProviderRepository, encryptionKey []byte) *CredentialProviderService {
	return &CredentialProviderService{repo: repo, encryptionKey: encryptionKey}
}

func (s *CredentialProviderService) SetLimiter(l *PlanLimiter) { s.limiter = l }

func (s *CredentialProviderService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *CredentialProviderService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.CredentialProvider, json.RawMessage, error) {
	cp, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, nil, err
	}
	if cp == nil {
		return nil, nil, domain.ErrNotFound
	}

	var config json.RawMessage
	if len(cp.ConfigEncrypted) > 0 {
		decrypted, err := crypto.Decrypt(cp.ConfigEncrypted, s.encryptionKey)
		if err != nil {
			config = json.RawMessage("{}")
		} else {
			config = json.RawMessage(decrypted)
		}
	} else {
		config = json.RawMessage("{}")
	}

	return cp, config, nil
}

func (s *CredentialProviderService) Create(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider, configJSON json.RawMessage) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, ResourceCredentialProviders); err != nil {
			return err
		}
	}
	normalizeCredentialProvider(cp)
	if err := validateCredentialProvider(cp); err != nil {
		return err
	}

	encrypted, err := crypto.Encrypt(configJSON, s.encryptionKey)
	if err != nil {
		return domain.NewValidationError("config", "failed to encrypt configuration")
	}
	cp.ConfigEncrypted = encrypted

	return s.repo.Create(ctx, tenantID, cp)
}

func (s *CredentialProviderService) Update(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider, configJSON json.RawMessage) error {
	normalizeCredentialProvider(cp)
	if err := validateCredentialProvider(cp); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, cp.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}

	encrypted, err := crypto.Encrypt(configJSON, s.encryptionKey)
	if err != nil {
		return domain.NewValidationError("config", "failed to encrypt configuration")
	}
	cp.ConfigEncrypted = encrypted

	return s.repo.Update(ctx, tenantID, cp)
}

func (s *CredentialProviderService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeCredentialProvider(cp *domain.CredentialProvider) {
	cp.Name = strings.TrimSpace(cp.Name)
	cp.Description = strings.TrimSpace(cp.Description)
}

func validateCredentialProvider(cp *domain.CredentialProvider) error {
	if cp.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if cp.Type == "" {
		return domain.NewValidationError("type", "credential type is required")
	}
	return nil
}
