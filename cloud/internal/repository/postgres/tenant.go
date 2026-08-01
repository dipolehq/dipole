package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TenantRepo struct {
	pool Pool
}

func (r *TenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	t.ID = uuid.New()
	if t.Plan == "" {
		t.Plan = "starter"
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, subdomain, plan, api_token_hash) VALUES ($1, $2, $3, $4, $5)`,
		t.ID, t.Name, t.Subdomain, t.Plan, t.APITokenHash)
	if err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	return nil
}

func (r *TenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, subdomain, plan, api_token_hash, created_at, updated_at FROM tenants WHERE id = $1`, id).
		Scan(&t.ID, &t.Name, &t.Subdomain, &t.Plan, &t.APITokenHash, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, subdomain, plan, api_token_hash, created_at, updated_at FROM tenants WHERE subdomain = $1`, subdomain).
		Scan(&t.ID, &t.Name, &t.Subdomain, &t.Plan, &t.APITokenHash, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by subdomain: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) GetByAPITokenHash(ctx context.Context, hash string) (*domain.Tenant, error) {
	if hash == "" {
		return nil, nil
	}
	t := &domain.Tenant{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, subdomain, plan, api_token_hash, created_at, updated_at FROM tenants WHERE api_token_hash = $1`, hash).
		Scan(&t.ID, &t.Name, &t.Subdomain, &t.Plan, &t.APITokenHash, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by api token hash: %w", err)
	}
	return t, nil
}

func (r *TenantRepo) List(ctx context.Context, opts domain.ListOptions) ([]domain.Tenant, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, subdomain, plan, api_token_hash, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Subdomain, &t.Plan, &t.APITokenHash, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

func (r *TenantRepo) Update(ctx context.Context, t *domain.Tenant) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE tenants SET name = $1, subdomain = $2, plan = $3, api_token_hash = $4, updated_at = NOW() WHERE id = $5`,
		t.Name, t.Subdomain, t.Plan, t.APITokenHash, t.ID)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	return nil
}

func (r *TenantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	return nil
}
