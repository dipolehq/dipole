// password_reset.go implements the "forgot password" flow: token generation, emailing, and password reset.
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
	Token   string `json:"token"`
	Message string `json:"message"`
}

const genericForgotPasswordMessage = "If an account with that email exists, a password reset link has been sent."

func (s *PasswordResetService) ForgotPassword(ctx context.Context, req *ForgotPasswordRequest) (*ForgotPasswordResponse, error) {
	if req.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	tenantSummaries, err := s.users.GetTenantsByEmail(ctx, req.Email)
	if err != nil || len(tenantSummaries) == 0 {
		return &ForgotPasswordResponse{Message: genericForgotPasswordMessage}, nil
	}

	var firstToken string
	for _, ts := range tenantSummaries {
		user, err := s.users.GetByEmail(ctx, ts.ID, req.Email)
		if err != nil || user == nil || !user.IsActive {
			continue
		}

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
	if len(req.NewPassword) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
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
