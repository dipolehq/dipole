/*
auth_login.go implements email+password authentication for admin users.

LoginService validates credentials, generates session tokens (JWTs), and
provides the POST /api/v1/auth/login endpoint logic. Passwords are hashed
with bcrypt. Session tokens are short-lived JWTs signed by the Identity Issuer.
*/
package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type LoginService struct {
	users    repository.UserRepository
	tenants  repository.TenantRepository
	mfaRepo  repository.UserMFARepository
	issuer   *identity.IdentityIssuer
	encKey   []byte
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

// SetMFA injects MFA dependencies (called after construction to avoid circular setup).
func (s *LoginService) SetMFA(mfaRepo repository.UserMFARepository, encKey []byte) {
	s.mfaRepo = mfaRepo
	s.encKey = encKey
}

type LoginRequest struct {
	Email    string
	Password string
}

type LoginResponse struct {
	Token       string    `json:"token,omitempty"`
	ExpiresAt   time.Time `json:"expiresAt,omitempty"`
	User        LoginUser `json:"user,omitempty"`
	MFARequired bool      `json:"mfaRequired,omitempty"`
	MFAToken    string    `json:"mfaToken,omitempty"`
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

	// Check if MFA is enabled for this user.
	if s.mfaRepo != nil {
		mfa, _ := s.mfaRepo.GetByUserID(ctx, s.tenantID, user.ID)
		if mfa != nil && mfa.IsEnabled {
			// Issue a short-lived MFA token (5 min, limited scope).
			mfaToken, err := s.issuer.IssueToken(identity.IssuanceRequest{
				Subject:   user.Email,
				Audience:  "dipole-mfa",
				Lifetime:  5 * time.Minute,
				Algorithm: "ES256",
				CustomClaims: map[string]interface{}{
					"tenant_id": s.tenantID.String(),
					"user_id":   user.ID.String(),
					"purpose":   "mfa_challenge",
				},
			})
			if err != nil {
				return nil, fmt.Errorf("issue mfa token: %w", err)
			}
			return &LoginResponse{
				MFARequired: true,
				MFAToken:    mfaToken,
			}, nil
		}
	}

	return s.issueSessionToken(user)
}

// VerifyMFA validates a TOTP code (or backup code) and issues the session JWT.
func (s *LoginService) VerifyMFA(ctx context.Context, mfaToken string, code string) (*LoginResponse, error) {
	if mfaToken == "" || code == "" {
		return nil, fmt.Errorf("mfaToken and code are required")
	}

	// Validate the MFA token.
	if err := s.issuer.VerifyTokenSignature(mfaToken); err != nil {
		return nil, fmt.Errorf("invalid mfa token")
	}

	// Extract claims.
	claims, err := s.extractMFAClaims(mfaToken)
	if err != nil {
		return nil, err
	}

	if claims.Purpose != "mfa_challenge" {
		return nil, fmt.Errorf("invalid mfa token")
	}
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("mfa token expired")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid mfa token")
	}

	// Try TOTP code first.
	mfaSvc := NewMFAService(s.mfaRepo, s.encKey, s.tenantID)
	if err := mfaSvc.ValidateCode(ctx, userID, code); err != nil {
		// Try backup code.
		if bcErr := mfaSvc.UseBackupCode(ctx, userID, code); bcErr != nil {
			return nil, fmt.Errorf("invalid verification code")
		}
	}

	// MFA passed — issue full session token.
	user, err := s.users.GetByID(ctx, s.tenantID, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueSessionToken(user)
}

func (s *LoginService) issueSessionToken(user *domain.User) (*LoginResponse, error) {
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

type mfaClaims struct {
	TenantID string
	UserID   string
	Purpose  string
	Exp      int64
}

func (s *LoginService) extractMFAClaims(token string) (*mfaClaims, error) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid token format")
	}
	payload := parts[1]
	if pad := len(payload) % 4; pad != 0 {
		payload += strings.Repeat("=", 4-pad)
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("decode token: %w", err)
	}
	var claims struct {
		TenantID string `json:"tenant_id"`
		UserID   string `json:"user_id"`
		Purpose  string `json:"purpose"`
		Exp      int64  `json:"exp"`
	}
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("parse token claims: %w", err)
	}
	return &mfaClaims{
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
		Purpose:  claims.Purpose,
		Exp:      claims.Exp,
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
