// signup.go implements self-serve tenant provisioning: creates a tenant, admin user, and default resource set.
package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

const (
	subdomainMin = 100000
	subdomainMax = 999999
)

type SignupService struct {
	tenants        repository.TenantRepository
	users          repository.UserRepository
	roles          repository.RoleRepository
	resourceSets   repository.ResourceSetRepository
	tenantProfiles TenantProfileWriter
	issuer         *identity.IdentityIssuer
}

type TenantProfileWriter interface {
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.TenantProfile) error
}

func NewSignupService(
	tenants repository.TenantRepository,
	users repository.UserRepository,
	roles repository.RoleRepository,
	resourceSets repository.ResourceSetRepository,
	tenantProfiles TenantProfileWriter,
	issuer *identity.IdentityIssuer,
) *SignupService {
	return &SignupService{
		tenants:        tenants,
		users:          users,
		roles:          roles,
		resourceSets:   resourceSets,
		tenantProfiles: tenantProfiles,
		issuer:         issuer,
	}
}

type SignupRequest struct {
	FirstName   string
	LastName    string
	CompanyName string
	Email       string
	Password    string
}

type SignupResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	TenantID  string    `json:"tenantId"`
	Subdomain string    `json:"subdomain"`
	User      LoginUser `json:"user"`
}

func (s *SignupService) Signup(ctx context.Context, req *SignupRequest) (*SignupResponse, error) {
	if req.CompanyName == "" {
		return nil, fmt.Errorf("company name is required")
	}
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if req.FirstName == "" {
		return nil, fmt.Errorf("first name is required")
	}
	if req.LastName == "" {
		return nil, fmt.Errorf("last name is required")
	}
	if len(req.Password) < minPasswordLength {
		return nil, fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	subdomain, err := s.generateUniqueSubdomain(ctx)
	if err != nil {
		return nil, fmt.Errorf("generate tenant subdomain: %w", err)
	}

	apiToken, err := GenerateAPIToken()
	if err != nil {
		return nil, fmt.Errorf("generate api token: %w", err)
	}
	apiTokenHash := HashAPIToken(apiToken)

	tenant := &domain.Tenant{
		Name:         req.CompanyName,
		Subdomain:    subdomain,
		Plan:         "starter",
		APITokenHash: apiTokenHash,
	}
	if err := s.tenants.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	defaultRS := &domain.ResourceSet{
		ID:          uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff"),
		Name:        "Default",
		Description: "Default Resource Set",
	}
	if err := s.resourceSets.Create(ctx, tenant.ID, defaultRS); err != nil {
		return nil, fmt.Errorf("create default resource set: %w", err)
	}

	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	displayName := req.FirstName + " " + req.LastName
	user := &domain.User{
		Email:        req.Email,
		Name:         displayName,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		PasswordHash: passwordHash,
	}

	superAdminRole := &domain.Role{
		Name:        "SuperAdmin",
		Description: "Full administrative access",
		IsActive:    true,
	}
	if err := s.roles.Create(ctx, tenant.ID, superAdminRole); err != nil {
		return nil, fmt.Errorf("create super admin role: %w", err)
	}
	user.RoleID = &superAdminRole.ID

	if err := s.users.Create(ctx, tenant.ID, user); err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}

	_ = s.tenantProfiles.Upsert(ctx, tenant.ID, &domain.TenantProfile{
		TenantID:  tenant.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
	})

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-admin",
		Lifetime:  24 * time.Hour,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id":        tenant.ID.String(),
			"tenant_subdomain": tenant.Subdomain,
			"user_id":          user.ID.String(),
			"email":            user.Email,
			"name":             user.Name,
			"role_name":        "SuperAdmin",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("issue session token: %w", err)
	}

	return &SignupResponse{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		TenantID:  tenant.ID.String(),
		Subdomain: subdomain,
		User: LoginUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
		},
	}, nil
}

func (s *SignupService) generateUniqueSubdomain(ctx context.Context) (string, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		n, err := rand.Int(rand.Reader, big.NewInt(subdomainMax-subdomainMin+1))
		if err != nil {
			return "", fmt.Errorf("generate random subdomain: %w", err)
		}
		candidate := fmt.Sprintf("%d", subdomainMin+n.Int64())

		existing, err := s.tenants.GetBySubdomain(ctx, candidate)
		if err != nil {
			return "", fmt.Errorf("check subdomain availability: %w", err)
		}
		if existing == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not allocate a unique tenant subdomain after %d attempts", maxAttempts)
}
