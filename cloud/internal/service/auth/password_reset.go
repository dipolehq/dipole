/*
password_reset.go implements the "forgot password" flow:

  - ForgotPassword: generates a cryptographically random token, hashes it
    (SHA-256), stores the hash with a 1-hour expiry, and returns the raw
    token. In production this token would be emailed; for now it's returned
    in the API response for dev purposes.

  - ResetPassword: validates the raw token (hash-matches a non-expired,
    unused record), updates the user's password, and marks the token used.
*/
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

const (
	resetTokenExpiry = 1 * time.Hour
	resetTokenPrefix = "dipole_rst_"
)

// PasswordResetService is stateless with respect to tenant. ForgotPassword
// resolves the tenant(s) for the given email via GetTenantsByEmail (same
// cross-tenant lookup LoginService uses), and ResetPassword derives the
// tenant directly from the stored token record — so a single instance
// correctly serves every tenant, not just one baked in at startup.
type PasswordResetService struct {
	users        repository.UserRepository
	resetRepo    repository.PasswordResetTokenRepository
	emailSvc     service.EmailService
	resetBaseURL string
}

func NewPasswordResetService(
	users repository.UserRepository,
	resetRepo repository.PasswordResetTokenRepository,
) *PasswordResetService {
	return &PasswordResetService{
		users:     users,
		resetRepo: resetRepo,
	}
}

func (s *PasswordResetService) SetEmail(emailSvc service.EmailService, resetBaseURL string) {
	s.emailSvc = emailSvc
	s.resetBaseURL = resetBaseURL
}

type ForgotPasswordRequest struct {
	Email string
}

type ForgotPasswordResponse struct {
	// Token is returned in dev mode. In production, this would be sent via email.
	Token   string `json:"token"`
	Message string `json:"message"`
}

const genericForgotPasswordMessage = "If an account with that email exists, a password reset link has been sent."

func (s *PasswordResetService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	// An email may have an active account in more than one tenant (same
	// pattern as LoginService.Login). Issue a separate reset token scoped
	// to each tenant the user belongs to, so resetting the password in one
	// tenant never leaks into or affects another.
	tenantSummaries, err := s.users.GetTenantsByEmail(ctx, req.Email)
	if err != nil || len(tenantSummaries) == 0 {
		// Don't reveal whether the email exists — always return success message.
		return &ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil
	}

	var firstToken string
	for _, ts := range tenantSummaries {
		user, err := s.users.GetByEmail(ctx, ts.ID, req.Email)
		if err != nil || user == nil || !user.IsActive {
			continue
		}

		// Clean up old expired/used tokens for this user.
		_ = s.resetRepo.DeleteExpiredForUser(ctx, ts.ID, user.ID)

		rawToken, err := generateResetToken()
		if err != nil {
			return nil, fmt.Errorf("generate reset token: %w", err)
		}
		tokenHash := hashResetToken(rawToken)

		resetToken := &domain.PasswordResetToken{
			TenantID:  ts.ID,
			UserID:    user.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().UTC().Add(resetTokenExpiry),
		}
		if err := s.resetRepo.Create(ctx, ts.ID, resetToken); err != nil {
			return nil, fmt.Errorf("store reset token: %w", err)
		}

		if firstToken == "" {
			firstToken = rawToken
		}

		if s.emailSvc != nil {
			resetLink := s.resetBaseURL + "/reset-password?token=" + rawToken
			htmlBody := fmt.Sprintf(
				`<h2>Reset Your Password</h2>
				<p>Click the link below to reset your password. This link expires in 1 hour.</p>
				<p><a href="%s" style="display:inline-block;padding:12px 24px;background:#f97316;color:#fff;text-decoration:none;border-radius:6px;">Reset Password</a></p>
				<p style="color:#666;font-size:12px;">If you didn't request this, you can safely ignore this email.</p>`,
				resetLink,
			)
			if err := s.emailSvc.Send(ctx, req.Email, "Reset your Dipole password", htmlBody); err != nil {
				fmt.Printf("[EMAIL ERROR] %v\n", err)
			}
		}
	}

	if s.emailSvc != nil || firstToken == "" {
		return &ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil
	}

	// Dev mode (no email service configured): return the token directly.
	// If the email matched multiple tenants, only the first tenant's token
	// is returned here — this dev-only path is a single-token API response,
	// which is a fine limitation since production always uses email delivery.
	return &ForgotPasswordResponse{
		Token:   firstToken,
		Message: genericForgotPasswordMessage,
	}, nil
}

type ResetPasswordRequest struct {
	Token       string
	NewPassword string
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, req *ResetPasswordRequest) error {
	if req.Token == "" {
		return fmt.Errorf("reset token is required")
	}
	if req.NewPassword == "" {
		return fmt.Errorf("new password is required")
	}
	if len(req.NewPassword) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}

	tokenHash := hashResetToken(req.Token)

	record, err := s.resetRepo.GetByHash(ctx, tokenHash)
	if err != nil || record == nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	if record.Used {
		return fmt.Errorf("reset token has already been used")
	}

	if time.Now().UTC().After(record.ExpiresAt) {
		return fmt.Errorf("reset token has expired")
	}

	// The token record itself carries the tenant it was issued for
	// (set in ForgotPassword) — that's the authoritative tenant scope for
	// this reset, not a value baked into the service at construction.
	user, err := s.users.GetByID(ctx, record.TenantID, record.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	passwordHash, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	if err := s.users.Update(ctx, record.TenantID, user); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	// Mark token as used.
	if err := s.resetRepo.MarkUsed(ctx, record.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	return nil
}

func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return resetTokenPrefix + base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashResetToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
