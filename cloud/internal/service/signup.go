/*
signup.go implements self-serve tenant provisioning for new customers.

Creates a tenant (with unique subdomain), admin user, default resource set,
and issues a session JWT for immediate login after signup.
*/
package service

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

// subdomainMin/subdomainMax bound the random numeric tenant subdomain
// (e.g. "64275"). Kept in a fixed 6-digit range so URLs like
// https://64275-mcp.dipolehq.com are predictable in length and never
// reveal the customer's company name.
const (
	subdomainMin = 100000
	subdomainMax = 999999
)

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
			"tenant_id":        tenant.ID.String(),
			"tenant_subdomain": tenant.Subdomain,
			"user_id":          user.ID.String(),
			"email":            user.Email,
			"name":             user.Name,
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

// generateUniqueSubdomain picks a random 6-digit numeric subdomain (e.g.
// "64275") and retries on the rare collision. The subdomain is opaque by
// design — it never encodes the customer's company name, so tenant URLs
// (https://<subdomain>-mcp.dipolehq.com, etc.) don't leak who the customer
// is or let anyone probe for a specific company's tenant.
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
