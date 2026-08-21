// user_mfa.go implements the UserMFARepository for PostgreSQL. TOTP secrets and backup codes are stored encrypted (AES-256-GCM).
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type UserMFARepo struct {
	pool Pool
}

func (r *UserMFARepo) Create(ctx context.Context, tenantID uuid.UUID, mfa *domain.UserMFA) error {
	if mfa.ID == uuid.Nil {
		mfa.ID = uuid.New()
	}
	now := time.Now().UTC()
	mfa.CreatedAt = now
	mfa.UpdatedAt = now

	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_mfa (id, tenant_id, user_id, totp_secret_encrypted, is_enabled, backup_codes_encrypted, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		mfa.ID, tenantID, mfa.UserID, mfa.TOTPSecretEncrypted, mfa.IsEnabled, mfa.BackupCodesEncrypted, mfa.CreatedAt, mfa.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create user mfa: %w", err)
	}
	return nil
}

func (r *UserMFARepo) GetByUserID(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) (*domain.UserMFA, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, totp_secret_encrypted, is_enabled, backup_codes_encrypted, created_at, updated_at
		 FROM user_mfa
		 WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	)

	var m domain.UserMFA
	err := row.Scan(&m.ID, &m.TenantID, &m.UserID, &m.TOTPSecretEncrypted, &m.IsEnabled, &m.BackupCodesEncrypted, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user mfa: %w", err)
	}
	return &m, nil
}

func (r *UserMFARepo) Update(ctx context.Context, tenantID uuid.UUID, mfa *domain.UserMFA) error {
	mfa.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE user_mfa SET totp_secret_encrypted = $1, is_enabled = $2, backup_codes_encrypted = $3, updated_at = $4
		 WHERE tenant_id = $5 AND user_id = $6`,
		mfa.TOTPSecretEncrypted, mfa.IsEnabled, mfa.BackupCodesEncrypted, mfa.UpdatedAt, tenantID, mfa.UserID,
	)
	if err != nil {
		return fmt.Errorf("update user mfa: %w", err)
	}
	return nil
}

func (r *UserMFARepo) Delete(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_mfa WHERE tenant_id = $1 AND user_id = $2`,
		tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete user mfa: %w", err)
	}
	return nil
}
