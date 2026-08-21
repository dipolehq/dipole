// users.go implements business logic and validation for Users.
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type UserService struct {
	repo    repository.UserRepository
	limiter *service.PlanLimiter
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *UserService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.User, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *UserService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (s *UserService) GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error) {
	return s.repo.GetByEmail(ctx, tenantID, email)
}

func (s *UserService) Create(ctx context.Context, tenantID uuid.UUID, u *domain.User) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceUsers); err != nil {
			return err
		}
	}
	normalizeUser(u)
	if err := validateUser(u); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, u)
}

func (s *UserService) Update(ctx context.Context, tenantID uuid.UUID, u *domain.User) error {
	normalizeUser(u)
	if err := validateUser(u); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, u.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, u)
}

func (s *UserService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeUser(u *domain.User) {
	u.Email = strings.TrimSpace(u.Email)
	u.FirstName = strings.TrimSpace(u.FirstName)
	u.LastName = strings.TrimSpace(u.LastName)
	u.Phone = strings.TrimSpace(u.Phone)
}

func validateUser(u *domain.User) error {
	if u.Email == "" {
		return domain.NewValidationError("email", "email is required")
	}
	if u.FirstName == "" {
		return domain.NewValidationError("firstName", "firstName is required")
	}
	return nil
}
