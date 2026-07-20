package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type SignOnPolicyRepo struct {
	pool Pool
}

func (r *SignOnPolicyRepo) Get(ctx context.Context, tenantID uuid.UUID) (*domain.SignOnPolicy, error) {
	p := &domain.SignOnPolicy{TenantID: tenantID}
	err := r.pool.QueryRow(ctx,
		`SELECT require_sso, require_mfa, updated_at FROM sign_on_policies WHERE tenant_id = $1`, tenantID).
		Scan(&p.RequireSSO, &p.RequireMFA, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		// Return defaults (both false)
		return p, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get sign-on policy: %w", err)
	}
	return p, nil
}

func (r *SignOnPolicyRepo) Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.SignOnPolicy) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sign_on_policies (tenant_id, require_sso, require_mfa, updated_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET require_sso = $2, require_mfa = $3, updated_at = NOW()`,
		tenantID, p.RequireSSO, p.RequireMFA)
	if err != nil {
		return fmt.Errorf("upsert sign-on policy: %w", err)
	}
	return nil
}
