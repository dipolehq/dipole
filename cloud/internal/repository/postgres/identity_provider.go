package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type IdentityProviderRepo struct {
	pool Pool
}

func (r *IdentityProviderRepo) Create(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error {
	idp.ID = uuid.New()
	idp.TenantID = tenantID
	config := idp.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}

	err := r.pool.QueryRow(ctx,
		`INSERT INTO identity_providers (id, tenant_id, name, description, is_active, type, config)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING created_at, updated_at`,
		idp.ID, tenantID, idp.Name, idp.Description, idp.IsActive, idp.Type, config).
		Scan(&idp.CreatedAt, &idp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create identity provider: %w", err)
	}
	return nil
}

func (r *IdentityProviderRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.IdentityProvider, error) {
	idp := &domain.IdentityProvider{}
	var config []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, created_at, updated_at
		 FROM identity_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&idp.ID, &idp.TenantID, &idp.Name, &idp.Description, &idp.IsActive, &idp.Type, &config, &idp.CreatedAt, &idp.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get identity provider: %w", err)
	}
	idp.Config = config
	return idp, nil
}

func (r *IdentityProviderRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.IdentityProvider, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM identity_providers
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count identity providers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, created_at, updated_at
		 FROM identity_providers
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list identity providers: %w", err)
	}
	defer rows.Close()

	var results []domain.IdentityProvider
	for rows.Next() {
		var idp domain.IdentityProvider
		var config []byte
		if err := rows.Scan(&idp.ID, &idp.TenantID, &idp.Name, &idp.Description, &idp.IsActive, &idp.Type, &config, &idp.CreatedAt, &idp.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan identity provider: %w", err)
		}
		idp.Config = config
		results = append(results, idp)
	}
	return results, total, nil
}

func (r *IdentityProviderRepo) Update(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error {
	config := idp.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE identity_providers SET name=$1, description=$2, is_active=$3, type=$4, config=$5, updated_at=NOW()
		 WHERE tenant_id = $6 AND id = $7`,
		idp.Name, idp.Description, idp.IsActive, idp.Type, config, tenantID, idp.ID)
	if err != nil {
		return fmt.Errorf("update identity provider: %w", err)
	}
	return nil
}

func (r *IdentityProviderRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM identity_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete identity provider: %w", err)
	}
	return nil
}
