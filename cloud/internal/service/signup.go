/*
signup.go implements self-serve tenant provisioning for new customers.

Creates a tenant (with unique subdomain), admin user, default resource set,
and issues a session JWT for immediate login after signup.
*/
package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

var subdomainRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,30}[a-z0-9]$`)

type SignupService struct {
	tenants      repository.TenantRepository
	users        repository.UserRepository
	resourceSets repository.ResourceSetRepository
	issuer       *identity.IdentityIssuer
}

func NewSignupService(
	tenants repository.TenantRepository,
	users repository.UserRepository,
	resourceSets repository.ResourceSetRepository,
	issuer *identity.IdentityIssuer,
) *SignupService {
	return &SignupService{
		tenants:      tenants,
		users:        users,
		resourceSets: resourceSets,
		issuer:       issuer,
	}
}

type SignupRequest struct {
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
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("password must be at least 6 characters")
	}

	subdomain := slugify(req.CompanyName)
	if !subdomainRegex.MatchString(subdomain) {
		return nil, fmt.Errorf("company name produces an invalid subdomain — use letters, numbers, and hyphens")
	}

	existing, err := s.tenants.GetBySubdomain(ctx, subdomain)
	if err != nil {
		return nil, fmt.Errorf("check subdomain availability: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("a tenant with this name already exists — try a different company name")
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

	user := &domain.User{
		Email:        req.Email,
		Name:         req.CompanyName + " Admin",
		FirstName:    "Admin",
		IsActive:     true,
		PasswordHash: passwordHash,
	}
	if err := s.users.Create(ctx, tenant.ID, user); err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-admin",
		Lifetime:  24 * time.Hour,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id": tenant.ID.String(),
			"user_id":   user.ID.String(),
			"email":     user.Email,
			"name":      user.Name,
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

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 32 {
		s = s[:32]
	}
	return s
}
