/*
auth_login.go implements email+password authentication for admin users.

LoginService validates credentials, generates session tokens (JWTs), and
provides the POST /api/v1/auth/login endpoint logic. Passwords are hashed
with bcrypt. Session tokens are short-lived JWTs signed by the Identity Issuer.
*/
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type LoginService struct {
	users    repository.UserRepository
	tenants  repository.TenantRepository
	issuer   *identity.IdentityIssuer
	tenantID uuid.UUID
}

func NewLoginService(users repository.UserRepository, tenants repository.TenantRepository, issuer *identity.IdentityIssuer, tenantID uuid.UUID) *LoginService {
	return &LoginService{
		users:    users,
		tenants:  tenants,
		issuer:   issuer,
		tenantID: tenantID,
	}
}

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      LoginUser `json:"user"`
}

type LoginUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (s *LoginService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	user, err := s.users.GetByEmail(ctx, s.tenantID, req.Email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-admin",
		Lifetime:  24 * time.Hour,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id": s.tenantID.String(),
			"user_id":   user.ID.String(),
			"email":     user.Email,
			"name":      user.Name,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("issue session token: %w", err)
	}

	return &LoginResponse{
		Token:     token,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		User: LoginUser{
			ID:    user.ID.String(),
			Email: user.Email,
			Name:  user.Name,
		},
	}, nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func (s *LoginService) SetPassword(ctx context.Context, userID uuid.UUID, password string) error {
	user, err := s.users.GetByID(ctx, s.tenantID, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	return s.users.Update(ctx, s.tenantID, user)
}
