/*
password_reset.go implements the "forgot password" flow:

  - ForgotPassword: generates a cryptographically random token, hashes it
    (SHA-256), stores the hash with a 1-hour expiry, and returns the raw
    token. In production this token would be emailed; for now it's returned
    in the API response for dev purposes.

  - ResetPassword: validates the raw token (hash-matches a non-expired,
    unused record), updates the user's password, and marks the token used.
*/
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const (
	resetTokenExpiry = 1 * time.Hour
	resetTokenPrefix = "dipole_rst_"
)

type PasswordResetService struct {
	users        repository.UserRepository
	resetRepo    repository.PasswordResetTokenRepository
	tenantID     uuid.UUID
	emailSvc     EmailService
	resetBaseURL string
}

func NewPasswordResetService(
	users repository.UserRepository,
	resetRepo repository.PasswordResetTokenRepository,
	tenantID uuid.UUID,
) *PasswordResetService {
	return &PasswordResetService{
		users:     users,
		resetRepo: resetRepo,
		tenantID:  tenantID,
	}
}

func (s *PasswordResetService) SetEmail(emailSvc EmailService, resetBaseURL string) {
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

func (s *PasswordResetService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	user, err := s.users.GetByEmail(ctx, s.tenantID, req.Email)
	if err != nil || user == nil {
		// Don't reveal whether the email exists — always return success message.
		return &ForgotPasswordResponse{
			Message: "If an account with that email exists, a password reset link has been sent.",
		}, nil
	}

	if !user.IsActive {
		return &ForgotPasswordResponse{
			Message: "If an account with that email exists, a password reset link has been sent.",
		}, nil
	}

	// Clean up old expired/used tokens for this user.
	_ = s.resetRepo.DeleteExpiredForUser(ctx, s.tenantID, user.ID)

	// Generate a cryptographically random token.
	rawToken, err := generateResetToken()
	if err != nil {
		return nil, fmt.Errorf("generate reset token: %w", err)
	}

	// Hash for storage (never store raw tokens).
	tokenHash := hashResetToken(rawToken)

	resetToken := &domain.PasswordResetToken{
		TenantID:  s.tenantID,
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(resetTokenExpiry),
	}

	if err := s.resetRepo.Create(ctx, s.tenantID, resetToken); err != nil {
		return nil, fmt.Errorf("store reset token: %w", err)
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

		return &ForgotPasswordResponse{
			Message: "If an account with that email exists, a password reset link has been sent.",
		}, nil
	}

	return &ForgotPasswordResponse{
		Token:   rawToken,
		Message: "If an account with that email exists, a password reset link has been sent.",
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

	// Verify tenant matches (defense in depth).
	if record.TenantID != s.tenantID {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Update password.
	user, err := s.users.GetByID(ctx, s.tenantID, record.UserID)
	if err != nil || user == nil {
		return fmt.Errorf("user not found")
	}

	passwordHash, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	user.PasswordHash = passwordHash
	if err := s.users.Update(ctx, s.tenantID, user); err != nil {
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
