// Package service — access_conditions.go implements business logic and
// validation for Access Conditions, sitting between transport and repository.
package service

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type AccessConditionService struct {
	repo repository.AccessConditionRepository
}

func NewAccessConditionService(repo repository.AccessConditionRepository) *AccessConditionService {
	return &AccessConditionService{repo: repo}
}

func (s *AccessConditionService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessCondition, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *AccessConditionService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.AccessCondition, error) {
	ac, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if ac == nil {
		return nil, domain.ErrNotFound
	}
	return ac, nil
}

func (s *AccessConditionService) Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	normalizeAccessCondition(ac)
	if err := validateAccessCondition(ac); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, ac)
}

func (s *AccessConditionService) Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error {
	normalizeAccessCondition(ac)
	if err := validateAccessCondition(ac); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, ac.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, ac)
}

func (s *AccessConditionService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeAccessCondition(ac *domain.AccessCondition) {
	ac.Name = strings.TrimSpace(ac.Name)
	ac.Description = strings.TrimSpace(ac.Description)
}

func validateAccessCondition(ac *domain.AccessCondition) error {
	if ac.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if ac.Type == "" {
		return domain.NewValidationError("type", "condition type is required")
	}
	return nil
}
