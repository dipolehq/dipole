/*
mfa.go implements Multi-Factor Authentication (TOTP + Backup Codes):

  - Setup: Generates a TOTP secret, encrypts it, stores it (not yet enabled).
    Returns the secret and otpauth:// URI for QR code generation.
  - Verify: Validates a 6-digit TOTP code against the stored secret.
    On success, marks MFA as enabled.
  - BackupCodes: Generates 10 single-use backup codes, encrypts and stores them.
  - UseBackupCode: Validates and consumes a backup code.
  - Reset: Deletes all MFA config for a user.
  - Status: Returns whether MFA is configured and enabled.
*/
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
)

const (
	mfaIssuerName    = "Dipole"
	backupCodeCount  = 10
	backupCodeLength = 8 // characters per code
)

type MFAService struct {
	mfaRepo  repository.UserMFARepository
	encKey   []byte
	tenantID uuid.UUID
}

func NewMFAService(mfaRepo repository.UserMFARepository, encKey []byte, tenantID uuid.UUID) *MFAService {
	return &MFAService{
		mfaRepo:  mfaRepo,
		encKey:   encKey,
		tenantID: tenantID,
	}
}

// MFAStatus returns the current MFA state for a user.
type MFAStatus struct {
	Configured     bool      `json:"configured"`
	Enabled        bool      `json:"enabled"`
	HasBackupCodes bool      `json:"hasBackupCodes"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
}

func (s *MFAService) GetStatus(ctx context.Context, userID uuid.UUID) (*MFAStatus, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("get mfa status: %w", err)
	}
	if mfa == nil {
		return &MFAStatus{Configured: false, Enabled: false}, nil
	}
	return &MFAStatus{
		Configured:     true,
		Enabled:        mfa.IsEnabled,
		HasBackupCodes: len(mfa.BackupCodesEncrypted) > 0,
		CreatedAt:      mfa.CreatedAt,
	}, nil
}

// SetupResponse is returned after TOTP setup (before verification).
type SetupResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// Setup generates a new TOTP secret and stores it (not yet enabled).
// If MFA is already configured, it replaces the existing secret (re-setup).
func (s *MFAService) Setup(ctx context.Context, userID uuid.UUID, userEmail string) (*SetupResponse, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfaIssuerName,
		AccountName: userEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	secretEncrypted, err := postgres.Encrypt([]byte(key.Secret()), s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt totp secret: %w", err)
	}

	existing, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing mfa: %w", err)
	}

	if existing != nil {
		// Replace secret, reset enabled state (user must re-verify).
		existing.TOTPSecretEncrypted = secretEncrypted
		existing.IsEnabled = false
		if err := s.mfaRepo.Update(ctx, s.tenantID, existing); err != nil {
			return nil, fmt.Errorf("update mfa: %w", err)
		}
	} else {
		mfa := &domain.UserMFA{
			TenantID:            s.tenantID,
			UserID:              userID,
			TOTPSecretEncrypted: secretEncrypted,
			IsEnabled:           false,
		}
		if err := s.mfaRepo.Create(ctx, s.tenantID, mfa); err != nil {
			return nil, fmt.Errorf("create mfa: %w", err)
		}
	}

	return &SetupResponse{
		Secret: key.Secret(),
		URI:    key.URL(),
	}, nil
}

// Verify validates a TOTP code and enables MFA if correct.
func (s *MFAService) Verify(ctx context.Context, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}

	secret, err := postgres.Decrypt(mfa.TOTPSecretEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secret)) {
		return fmt.Errorf("invalid verification code")
	}

	// Enable MFA.
	mfa.IsEnabled = true
	if err := s.mfaRepo.Update(ctx, s.tenantID, mfa); err != nil {
		return fmt.Errorf("enable mfa: %w", err)
	}

	return nil
}

// ValidateCode checks a TOTP code without changing state (used during login).
func (s *MFAService) ValidateCode(ctx context.Context, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}
	if !mfa.IsEnabled {
		return fmt.Errorf("MFA not enabled")
	}

	secret, err := postgres.Decrypt(mfa.TOTPSecretEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secret)) {
		return fmt.Errorf("invalid code")
	}

	return nil
}

// GenerateBackupCodes creates 10 single-use recovery codes.
func (s *MFAService) GenerateBackupCodes(ctx context.Context, userID uuid.UUID) ([]string, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil || mfa == nil {
		return nil, fmt.Errorf("MFA not configured — set up authenticator first")
	}

	codes := make([]string, backupCodeCount)
	for i := range codes {
		codes[i] = generateBackupCode()
	}

	// Store as JSON-encoded, encrypted.
	codesJSON, err := json.Marshal(codes)
	if err != nil {
		return nil, fmt.Errorf("marshal backup codes: %w", err)
	}

	encrypted, err := postgres.Encrypt(codesJSON, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt backup codes: %w", err)
	}

	mfa.BackupCodesEncrypted = encrypted
	if err := s.mfaRepo.Update(ctx, s.tenantID, mfa); err != nil {
		return nil, fmt.Errorf("store backup codes: %w", err)
	}

	return codes, nil
}

// UseBackupCode validates and consumes a backup code (used during login).
func (s *MFAService) UseBackupCode(ctx context.Context, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}
	if len(mfa.BackupCodesEncrypted) == 0 {
		return fmt.Errorf("no backup codes configured")
	}

	decrypted, err := postgres.Decrypt(mfa.BackupCodesEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt backup codes: %w", err)
	}

	var codes []string
	if err := json.Unmarshal(decrypted, &codes); err != nil {
		return fmt.Errorf("parse backup codes: %w", err)
	}

	// Find and remove the matching code (case-insensitive).
	normalizedInput := strings.TrimSpace(strings.ToLower(code))
	found := false
	remaining := make([]string, 0, len(codes))
	for _, c := range codes {
		if !found && strings.ToLower(c) == normalizedInput {
			found = true
			continue // consume it
		}
		remaining = append(remaining, c)
	}

	if !found {
		return fmt.Errorf("invalid backup code")
	}

	// Re-encrypt remaining codes.
	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return fmt.Errorf("marshal remaining codes: %w", err)
	}
	encrypted, err := postgres.Encrypt(remainingJSON, s.encKey)
	if err != nil {
		return fmt.Errorf("encrypt remaining codes: %w", err)
	}

	mfa.BackupCodesEncrypted = encrypted
	if err := s.mfaRepo.Update(ctx, s.tenantID, mfa); err != nil {
		return fmt.Errorf("update backup codes: %w", err)
	}

	return nil
}

// Reset removes all MFA configuration for a user.
func (s *MFAService) Reset(ctx context.Context, userID uuid.UUID) error {
	return s.mfaRepo.Delete(ctx, s.tenantID, userID)
}

// IsEnabled returns whether MFA is enabled for a user (used by login flow).
func (s *MFAService) IsEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, s.tenantID, userID)
	if err != nil {
		return false, err
	}
	if mfa == nil {
		return false, nil
	}
	return mfa.IsEnabled, nil
}

func generateBackupCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, backupCodeLength)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	for i := range buf {
		buf[i] = charset[buf[i]%byte(len(charset))]
	}
	// Format as xxxx-xxxx for readability.
	return string(buf[:4]) + "-" + string(buf[4:])
}
