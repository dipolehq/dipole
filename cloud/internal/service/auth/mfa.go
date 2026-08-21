// mfa.go implements TOTP-based multi-factor authentication: setup, verification, and backup codes.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const (
	mfaIssuerName    = "Dipole"
	backupCodeCount  = 10
	backupCodeLength = 8
)

type MFAService struct {
	mfaRepo repository.UserMFARepository
	encKey  []byte
}

func NewMFAService(mfaRepo repository.UserMFARepository, encKey []byte) *MFAService {
	return &MFAService{
		mfaRepo: mfaRepo,
		encKey:  encKey,
	}
}

type MFAStatus struct {
	Configured     bool      `json:"configured"`
	Enabled        bool      `json:"enabled"`
	HasBackupCodes bool      `json:"hasBackupCodes"`
	CreatedAt      time.Time `json:"createdAt,omitempty"`
}

func (s *MFAService) GetStatus(ctx context.Context, tenantID, userID uuid.UUID) (*MFAStatus, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
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

type SetupResponse struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

func (s *MFAService) Setup(ctx context.Context, tenantID, userID uuid.UUID, userEmail string) (*SetupResponse, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      mfaIssuerName,
		AccountName: userEmail,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	secretEncrypted, err := crypto.Encrypt([]byte(key.Secret()), s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt totp secret: %w", err)
	}

	existing, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		return nil, fmt.Errorf("check existing mfa: %w", err)
	}

	if existing != nil {
		existing.TOTPSecretEncrypted = secretEncrypted
		existing.IsEnabled = false
		if err := s.mfaRepo.Update(ctx, tenantID, existing); err != nil {
			return nil, fmt.Errorf("update mfa: %w", err)
		}
	} else {
		mfa := &domain.UserMFA{
			TenantID:            tenantID,
			UserID:              userID,
			TOTPSecretEncrypted: secretEncrypted,
			IsEnabled:           false,
		}
		if err := s.mfaRepo.Create(ctx, tenantID, mfa); err != nil {
			return nil, fmt.Errorf("create mfa: %w", err)
		}
	}

	return &SetupResponse{
		Secret: key.Secret(),
		URI:    key.URL(),
	}, nil
}

func (s *MFAService) Verify(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}

	secret, err := crypto.Decrypt(mfa.TOTPSecretEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secret)) {
		return fmt.Errorf("invalid verification code")
	}

	mfa.IsEnabled = true
	if err := s.mfaRepo.Update(ctx, tenantID, mfa); err != nil {
		return fmt.Errorf("enable mfa: %w", err)
	}

	return nil
}

func (s *MFAService) ValidateCode(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}
	if !mfa.IsEnabled {
		return fmt.Errorf("MFA not enabled")
	}

	secret, err := crypto.Decrypt(mfa.TOTPSecretEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secret)) {
		return fmt.Errorf("invalid code")
	}

	return nil
}

func (s *MFAService) GenerateBackupCodes(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil || mfa == nil {
		return nil, fmt.Errorf("MFA not configured — set up authenticator first")
	}

	codes := make([]string, backupCodeCount)
	for i := range codes {
		code, err := generateBackupCode()
		if err != nil {
			return nil, fmt.Errorf("generate backup code: %w", err)
		}
		codes[i] = code
	}

	codesJSON, err := json.Marshal(codes)
	if err != nil {
		return nil, fmt.Errorf("marshal backup codes: %w", err)
	}

	encrypted, err := crypto.Encrypt(codesJSON, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt backup codes: %w", err)
	}

	mfa.BackupCodesEncrypted = encrypted
	if err := s.mfaRepo.Update(ctx, tenantID, mfa); err != nil {
		return nil, fmt.Errorf("store backup codes: %w", err)
	}

	return codes, nil
}

func (s *MFAService) UseBackupCode(ctx context.Context, tenantID, userID uuid.UUID, code string) error {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil || mfa == nil {
		return fmt.Errorf("MFA not configured")
	}
	if len(mfa.BackupCodesEncrypted) == 0 {
		return fmt.Errorf("no backup codes configured")
	}

	decrypted, err := crypto.Decrypt(mfa.BackupCodesEncrypted, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt backup codes: %w", err)
	}

	var codes []string
	if err := json.Unmarshal(decrypted, &codes); err != nil {
		return fmt.Errorf("parse backup codes: %w", err)
	}

	normalizedInput := strings.TrimSpace(strings.ToLower(code))
	found := false
	remaining := make([]string, 0, len(codes))
	for _, c := range codes {
		if !found && strings.ToLower(c) == normalizedInput {
			found = true
			continue
		}
		remaining = append(remaining, c)
	}

	if !found {
		return fmt.Errorf("invalid backup code")
	}

	remainingJSON, err := json.Marshal(remaining)
	if err != nil {
		return fmt.Errorf("marshal remaining codes: %w", err)
	}
	encrypted, err := crypto.Encrypt(remainingJSON, s.encKey)
	if err != nil {
		return fmt.Errorf("encrypt remaining codes: %w", err)
	}

	mfa.BackupCodesEncrypted = encrypted
	if err := s.mfaRepo.Update(ctx, tenantID, mfa); err != nil {
		return fmt.Errorf("update backup codes: %w", err)
	}

	return nil
}

func (s *MFAService) Reset(ctx context.Context, tenantID, userID uuid.UUID) error {
	return s.mfaRepo.Delete(ctx, tenantID, userID)
}

func (s *MFAService) IsEnabled(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	mfa, err := s.mfaRepo.GetByUserID(ctx, tenantID, userID)
	if err != nil {
		return false, err
	}
	if mfa == nil {
		return false, nil
	}
	return mfa.IsEnabled, nil
}

func generateBackupCode() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	buf := make([]byte, backupCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand failed: %w", err)
	}
	for i := range buf {
		buf[i] = charset[buf[i]%byte(len(charset))]
	}
	return string(buf[:4]) + "-" + string(buf[4:]), nil
}
