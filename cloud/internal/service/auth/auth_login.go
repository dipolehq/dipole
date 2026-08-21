// auth_login.go implements email+password admin login, including tenant resolution, MFA challenge, and session token issuance.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type LoginService struct {
	users   repository.UserRepository
	tenants repository.TenantRepository
	roles   repository.RoleRepository
	mfaRepo repository.UserMFARepository
	mfaSvc  *MFAService
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

func (s *LoginService) SetMFA(mfaRepo repository.UserMFARepository, encKey []byte) {
	s.mfaRepo = mfaRepo
	s.encKey = encKey
	s.mfaSvc = NewMFAService(mfaRepo, encKey)
}

func (s *LoginService) SetRoles(roles repository.RoleRepository) {
	s.roles = roles
}

type LoginRequest struct {
	Email    string
	Password string
	TenantID string
}

type LoginOutcome int

const (
	LoginOutcomeSuccess LoginOutcome = iota
	LoginOutcomeMFARequired
	LoginOutcomeTenantSelectionRequired
)

type LoginResponse struct {
	Outcome LoginOutcome `json:"-"`
	loginSuccess
	loginMFAChallenge
	loginTenantChoice
}

type loginSuccess struct {
	Token           string    `json:"token,omitempty"`
	ExpiresAt       time.Time `json:"expiresAt,omitempty"`
	User            LoginUser `json:"user,omitempty"`
	TenantSubdomain string    `json:"tenantSubdomain,omitempty"`
}

type loginMFAChallenge struct {
	MFARequired bool   `json:"mfaRequired,omitempty"`
	MFAToken    string `json:"mfaToken,omitempty"`
}

type loginTenantChoice struct {
	Tenants []domain.TenantSummary `json:"tenants,omitempty"`
}

type LoginUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func (s *LoginService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	targetTenantID, tenantChoices, err := s.resolveTenantForLogin(ctx, req)
	if err != nil {
		return nil, err
	}
	if tenantChoices != nil {
		return &LoginResponse{
			Outcome:           LoginOutcomeTenantSelectionRequired,
			loginTenantChoice: loginTenantChoice{Tenants: tenantChoices},
		}, nil
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

	mfaToken, mfaRequired, err := s.checkMFAChallenge(ctx, targetTenantID, user)
	if err != nil {
		return nil, err
	}
	if mfaRequired {
		return &LoginResponse{
			Outcome: LoginOutcomeMFARequired,
			loginMFAChallenge: loginMFAChallenge{
				MFARequired: true,
				MFAToken:    mfaToken,
			},
		}, nil
	}

	return s.issueSessionTokenForTenant(ctx, user, targetTenantID)
}

func (s *LoginService) resolveTenantForLogin(ctx context.Context, req *LoginRequest) (tenantID uuid.UUID, tenants []domain.TenantSummary, err error) {
	if req.TenantID != "" {
		parsed, err := uuid.Parse(req.TenantID)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("invalid tenant selection")
		}
		return parsed, nil, nil
	}

	candidateTenants, err := s.users.GetTenantsByEmail(ctx, req.Email)
	if err != nil || len(candidateTenants) == 0 {
		return uuid.Nil, nil, fmt.Errorf("invalid email or password")
	}
	if len(candidateTenants) == 1 {
		return candidateTenants[0].ID, nil, nil
	}

	var user *domain.User
	for _, t := range candidateTenants {
		user, _ = s.users.GetByEmail(ctx, t.ID, req.Email)
		if user != nil {
			break
		}
	}
	if user == nil {
		return uuid.Nil, nil, fmt.Errorf("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return uuid.Nil, nil, fmt.Errorf("invalid email or password")
	}
	return uuid.Nil, candidateTenants, nil
}

func (s *LoginService) checkMFAChallenge(ctx context.Context, tenantID uuid.UUID, user *domain.User) (mfaToken string, required bool, err error) {
	if s.mfaRepo == nil {
		return "", false, nil
	}

	mfa, _ := s.mfaRepo.GetByUserID(ctx, tenantID, user.ID)
	if mfa == nil || !mfa.IsEnabled {
		return "", false, nil
	}

	mfaToken, err = s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-mfa",
		Lifetime:  5 * time.Minute,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id": tenantID.String(),
			"user_id":   user.ID.String(),
			"purpose":   "mfa_challenge",
		},
	})
	if err != nil {
		return "", false, fmt.Errorf("issue mfa token: %w", err)
	}
	return mfaToken, true, nil
}

func (s *LoginService) VerifyMFA(ctx context.Context, mfaToken string, code string) (*LoginResponse, error) {
	if mfaToken == "" || code == "" {
		return nil, fmt.Errorf("mfaToken and code are required")
	}

	if err := s.issuer.VerifyTokenSignature(mfaToken); err != nil {
		return nil, fmt.Errorf("invalid mfa token")
	}

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

	mfaTenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid mfa token")
	}

	if err := s.mfaSvc.ValidateCode(ctx, mfaTenantID, userID, code); err != nil {
		if bcErr := s.mfaSvc.UseBackupCode(ctx, mfaTenantID, userID, code); bcErr != nil {
			return nil, fmt.Errorf("invalid verification code")
		}
	}

	user, err := s.users.GetByID(ctx, mfaTenantID, userID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.issueSessionTokenForTenant(ctx, user, mfaTenantID)
}

func (s *LoginService) issueSessionTokenForTenant(ctx context.Context, user *domain.User, tenantID uuid.UUID) (*LoginResponse, error) {
	var tenantSubdomain string
	if tenant, err := s.tenants.GetByID(ctx, tenantID); err == nil && tenant != nil {
		tenantSubdomain = tenant.Subdomain
	}

	var roleName string
	if s.roles != nil && user.RoleID != nil {
		if role, err := s.roles.GetByID(ctx, tenantID, *user.RoleID); err == nil && role != nil {
			roleName = role.Name
		}
	}

	token, err := s.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   user.Email,
		Audience:  "dipole-admin",
		Lifetime:  30 * time.Minute,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id":        tenantID.String(),
			"tenant_subdomain": tenantSubdomain,
			"user_id":          user.ID.String(),
			"email":            user.Email,
			"name":             user.Name,
			"role_name":        roleName,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("issue session token: %w", err)
	}

	return &LoginResponse{
		Outcome: LoginOutcomeSuccess,
		loginSuccess: loginSuccess{
			Token:           token,
			ExpiresAt:       time.Now().UTC().Add(30 * time.Minute),
			TenantSubdomain: tenantSubdomain,
			User: LoginUser{
				ID:    user.ID.String(),
				Email: user.Email,
				Name:  user.Name,
				Role:  roleName,
			},
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
	raw, err := decodeJWTPayload(token)
	if err != nil {
		return nil, fmt.Errorf("invalid token format")
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

func (s *LoginService) SwitchTenant(ctx context.Context, email string, targetTenantID uuid.UUID) (*LoginResponse, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if targetTenantID == uuid.Nil {
		return nil, fmt.Errorf("target tenant is required")
	}

	user, err := s.users.GetByEmail(ctx, targetTenantID, email)
	if err != nil || user == nil {
		return nil, fmt.Errorf("access denied: no account in target tenant")
	}
	if !user.IsActive {
		return nil, fmt.Errorf("access denied: account is disabled in target tenant")
	}

	return s.issueSessionTokenForTenant(ctx, user, targetTenantID)
}

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

func (s *LoginService) ListTenantsForUser(ctx context.Context, email string) ([]domain.TenantSummary, error) {
	if email == "" {
		return nil, fmt.Errorf("email is required")
	}
	return s.users.GetTenantsByEmail(ctx, email)
}
