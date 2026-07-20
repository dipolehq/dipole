package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TenantProfileRepo struct {
	pool Pool
}

func (r *TenantProfileRepo) Get(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error) {
	p := &domain.TenantProfile{TenantID: tenantID}
	err := r.pool.QueryRow(ctx,
		`SELECT first_name, last_name, email, phone, updated_at FROM tenant_profiles WHERE tenant_id = $1`, tenantID).
		Scan(&p.FirstName, &p.LastName, &p.Email, &p.Phone, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant profile: %w", err)
	}
	return p, nil
}

func (r *TenantProfileRepo) Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.TenantProfile) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_profiles (tenant_id, first_name, last_name, email, phone, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (tenant_id) DO UPDATE SET first_name=$2, last_name=$3, email=$4, phone=$5, updated_at=NOW()`,
		tenantID, p.FirstName, p.LastName, p.Email, p.Phone)
	if err != nil {
		return fmt.Errorf("upsert tenant profile: %w", err)
	}
	return nil
}
