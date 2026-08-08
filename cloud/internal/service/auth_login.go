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

// LoginService is stateless with respect to tenant: Login resolves the
// target tenant per-call (via the caller-supplied tenantId, or by looking
// up all tenants for the given email), and every other method receives the
// tenant it should act on as an explicit parameter. This lets a single
// instance correctly serve every tenant.
type LoginService struct {
	users   repository.UserRepository
	tenants repository.TenantRepository
	mfaRepo repository.UserMFARepository
	issuer  *identity.IdentityIssuer
	encKey  []byte
}

func NewLoginService(users repository.UserRepository, tenants repository.TenantRepository, issuer *identity.IdentityIssuer) *LoginService {
	return &LoginService{
		users:   users,
		tenants: tenants,
		issuer:  issuer,
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
	TenantID string // Optional — if user has multiple tenants, they pick one
}

type LoginResponse struct {
	Token     string    `json:"token,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	User      LoginUser `json:"user,omitempty"`
	// TenantSubdomain is the authenticated tenant's subdomain. The admin console
	// needs it to display this tenant's per-tenant service URLs (MCP
	// Authorization Server, identity issuer, IdP callback) — those are values
	// the customer copies into their IdP, so showing another tenant's would
	// produce a redirect_uri mismatch.
	TenantSubdomain string                 `json:"tenantSubdomain,omitempty"`
	MFARequired     bool                   `json:"mfaRequired,omitempty"`
	MFAToken        string                 `json:"mfaToken,omitempty"`
	Tenants         []domain.TenantSummary `json:"tenants,omitempty"` // Returned when user must pick a tenant
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

	// Determine which tenant to authenticate against. There is no
	// service-level default — every login must resolve a real tenant
	// either from the caller's explicit selection or by looking up which
	// tenant(s) the email belongs to.
	var targetTenantID uuid.UUID

	if req.TenantID != "" {
		// User explicitly selected a tenant (from the picker).
		parsed, err := uuid.Parse(req.TenantID)
		if err != nil {
			return nil, fmt.Errorf("invalid tenant selection")
		}
		targetTenantID = parsed
	} else {
		tenants, err := s.users.GetTenantsByEmail(ctx, req.Email)
		if err != nil || len(tenants) == 0 {
			return nil, fmt.Errorf("invalid email or password")
		}
		if len(tenants) > 1 {
			// Email has an active account in multiple tenants — verify the
			// password against whichever tenant's account matches before
			// revealing the tenant list (avoids leaking tenant membership
			// to someone who doesn't actually know the password).
			var user *domain.User
			for _, t := range tenants {
				user, _ = s.users.GetByEmail(ctx, t.ID, req.Email)
				if user != nil {
					break
				}
			}
			if user == nil {
				return nil, fmt.Errorf("invalid email or password")
			}
			if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
				return nil, fmt.Errorf("invalid email or password")
			}
			return &LoginResponse{Tenants: tenants}, nil
		}
		targetTenantID = tenants[0].ID
	}

	user, err := s.users.GetByEmail(ctx, targetTenantID, req.Email)
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
		mfa, _ := s.mfaRepo.GetByUserID(ctx, targetTenantID, user.ID)
		if mfa != nil && mfa.IsEnabled {
			// Issue a short-lived MFA token (5 min, limited scope).
			mfaToken, err := s.issuer.IssueToken(identity.IssuanceRequest{
				Subject:   user.Email,
				Audience:  "dipole-mfa",
				Lifetime:  5 * time.Minute,
				Algorithm: "ES256",
				CustomClaims: map[string]interface{}{
					"tenant_id": targetTenantID.String(),
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

	return s.issueSessionTokenForTenant(ctx, user, targetTenantID)
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

	// The MFA challenge token carries the tenant the user is logging into
	// (set in Login when the token was issued) — use that, not the
	// service's construction-time default, so this works for any tenant.
	mfaTenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid mfa token")
	}

	// Try TOTP code first.
	mfaSvc := NewMFAService(s.mfaRepo, s.encKey)
	if err := mfaSvc.ValidateCode(ctx, mfaTenantID, userID, code); err != nil {
		// Try backup code.
		if bcErr := mfaSvc.UseBackupCode(ctx, mfaTenantID, userID, code); bcErr != nil {
			return nil, fmt.Errorf("invalid verification code")
		}
	}

	// MFA passed — issue full session token.
	user, err := s.users.GetByID(ctx, mfaTenantID, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueSessionTokenForTenant(ctx, user, mfaTenantID)
}

func (s *LoginService) issueSessionTokenForTenant(ctx context.Context, user *domain.User, tenantID uuid.UUID) (*LoginResponse, error) {
	// Best-effort: the console degrades to hiding per-tenant URLs rather than
	// failing login if this lookup doesn't succeed.
	var tenantSubdomain string
	if tenant, err := s.tenants.GetByID(ctx, tenantID); err == nil && tenant != nil {
		tenantSubdomain = tenant.Subdomain
	}

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-admin",
		Lifetime:  30 * time.Minute,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id": tenantID.String(),
			// Carried in the token so the console can derive this tenant's
			// per-tenant service URLs from the session itself, rather than from
			// separately-synced client state that can go stale or be missing.
			"tenant_subdomain": tenantSubdomain,
			"user_id":          user.ID.String(),
			"email":            user.Email,
			"name":             user.Name,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("issue session token: %w", err)
	}

	return &LoginResponse{
		Token:           token,
		ExpiresAt:       time.Now().UTC().Add(30 * time.Minute),
		TenantSubdomain: tenantSubdomain,
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

func (s *LoginService) SetPassword(ctx context.Context, tenantID, userID uuid.UUID, password string) error {
	user, err := s.users.GetByID(ctx, tenantID, userID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	return s.users.Update(ctx, tenantID, user)
}

// SwitchTenant issues a new session JWT scoped to a different tenant.
// The caller must already be authenticated. We verify the user's email
// has an active record in the target tenant to prevent unauthorized access.
func (s *LoginService) SwitchTenant(ctx context.Context, email string, targetTenantID uuid.UUID) (*LoginResponse, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if targetTenantID == uuid.Nil {
		return nil, fmt.Errorf("target tenant is required")
	}

	// Verify the user has an active account in the target tenant.
	user, err := s.users.GetByEmail(ctx, targetTenantID, email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("access denied: no account in target tenant")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("access denied: account is disabled in target tenant")
	}

	// Reuse the shared issuance path so the response (including the tenant
	// subdomain the console needs) is identical to a fresh login.
	return s.issueSessionTokenForTenant(ctx, user, targetTenantID)
}

// RefreshSession issues a fresh session token for an already-authenticated user.
// Called by the frontend every 5 minutes to extend the session without re-login.
func (s *LoginService) RefreshSession(ctx context.Context, email string, tenantID uuid.UUID) (*LoginResponse, error) {
	if email == "" || tenantID == uuid.Nil {
		return nil, fmt.Errorf("invalid session")
	}

	user, err := s.users.GetByEmail(ctx, tenantID, email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	return s.issueSessionTokenForTenant(ctx, user, tenantID)
}

// ListTenantsForUser returns all tenants where the given email has an active account.
func (s *LoginService) ListTenantsForUser(ctx context.Context, email string) ([]domain.TenantSummary, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	return s.users.GetTenantsByEmail(ctx, email)
}
