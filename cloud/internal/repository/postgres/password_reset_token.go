/*
password_reset_token.go implements the PasswordResetTokenRepository for PostgreSQL.
Tokens are stored with SHA-256 hashes and 1-hour expiry. Once consumed, they
are marked used (not deleted) to prevent replay attacks.
*/
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type PasswordResetTokenRepo struct {
	pool Pool
}

func (r *PasswordResetTokenRepo) Create(ctx context.Context, tenantID uuid.UUID, token *domain.PasswordResetToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, tenant_id, user_id, token_hash, expires_at, used, created_at)
		 VALUES ($1, $2, $3, $4, $5, false, now())`,
		token.ID, tenantID, token.UserID, token.TokenHash, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.PasswordResetToken, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, token_hash, expires_at, used, created_at
		 FROM password_reset_tokens
		 WHERE token_hash = $1`,
		tokenHash,
	)

	var t domain.PasswordResetToken
	var createdAt time.Time
	err := row.Scan(&t.ID, &t.TenantID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Used, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("get password reset token: %w", err)
	}
	t.CreatedAt = createdAt
	return &t, nil
}

func (r *PasswordResetTokenRepo) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used = true WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("mark password reset token used: %w", err)
	}
	return nil
}

func (r *PasswordResetTokenRepo) DeleteExpiredForUser(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM password_reset_tokens WHERE tenant_id = $1 AND user_id = $2 AND (used = true OR expires_at < now())`,
		tenantID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete expired reset tokens: %w", err)
	}
	return nil
}
