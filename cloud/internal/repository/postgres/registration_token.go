/*
registration_token.go implements the RegistrationTokenRepository for PostgreSQL.

Stores SHA-256 hashes of proxy registration tokens with expiry and consumption
tracking. Tokens are marked consumed (not deleted) to maintain an audit trail.
*/
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type RegistrationTokenRepo struct {
	pool Pool
}

func (r *RegistrationTokenRepo) Store(ctx context.Context, tenantID uuid.UUID, tokenHash string, controllerID uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO proxy_registration_tokens (id, tenant_id, token_hash, controller_id, consumed, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, false, $5, now())`,
		uuid.New(), tenantID, tokenHash, controllerID, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store registration token: %w", err)
	}
	return nil
}

func (r *RegistrationTokenRepo) Consume(ctx context.Context, tenantID uuid.UUID, tokenHash string) (*domain.RegistrationToken, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE proxy_registration_tokens
		 SET consumed = true
		 WHERE tenant_id = $1 AND token_hash = $2 AND consumed = false AND expires_at > now()
		 RETURNING id, tenant_id, token_hash, controller_id, consumed, expires_at, created_at`,
		tenantID, tokenHash,
	)

	t := &domain.RegistrationToken{}
	err := row.Scan(&t.ID, &t.TenantID, &t.TokenHash, &t.ControllerID, &t.Consumed, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("consume registration token: %w", err)
	}
	return t, nil
}
