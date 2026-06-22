package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type CredentialProviderRepo struct {
	pool Pool
}

func (r *CredentialProviderRepo) Create(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider) error {
	cp.ID = uuid.New()
	cp.TenantID = tenantID
	tags, _ := json.Marshal(cp.Tags)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO credential_providers (id, tenant_id, name, description, is_active, type, config_encrypted, tags, integration_id, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		cp.ID, tenantID, cp.Name, cp.Description, cp.IsActive, cp.Type, cp.ConfigEncrypted, tags, cp.IntegrationID, cp.ResourceSetID)
	if err != nil {
		return fmt.Errorf("create credential provider: %w", err)
	}
	return nil
}

func (r *CredentialProviderRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CredentialProvider, error) {
	cp := &domain.CredentialProvider{}
	var tags []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config_encrypted, tags, integration_id, resource_set_id, created_at, updated_at
		 FROM credential_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&cp.ID, &cp.TenantID, &cp.Name, &cp.Description, &cp.IsActive, &cp.Type, &cp.ConfigEncrypted, &tags, &cp.IntegrationID, &cp.ResourceSetID, &cp.CreatedAt, &cp.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get credential provider: %w", err)
	}

	json.Unmarshal(tags, &cp.Tags)
	return cp, nil
}

func (r *CredentialProviderRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM credential_providers WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count credential providers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config_encrypted, tags, integration_id, resource_set_id, created_at, updated_at
		 FROM credential_providers WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list credential providers: %w", err)
	}
	defer rows.Close()

	var results []domain.CredentialProvider
	for rows.Next() {
		var cp domain.CredentialProvider
		var tags []byte
		if err := rows.Scan(&cp.ID, &cp.TenantID, &cp.Name, &cp.Description, &cp.IsActive, &cp.Type, &cp.ConfigEncrypted, &tags, &cp.IntegrationID, &cp.ResourceSetID, &cp.CreatedAt, &cp.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan credential provider: %w", err)
		}
		json.Unmarshal(tags, &cp.Tags)
		results = append(results, cp)
	}
	return results, total, nil
}

func (r *CredentialProviderRepo) Update(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider) error {
	tags, _ := json.Marshal(cp.Tags)

	_, err := r.pool.Exec(ctx,
		`UPDATE credential_providers SET name=$1, description=$2, is_active=$3, type=$4, config_encrypted=$5, tags=$6, integration_id=$7, resource_set_id=$8, updated_at=NOW()
		 WHERE tenant_id = $9 AND id = $10`,
		cp.Name, cp.Description, cp.IsActive, cp.Type, cp.ConfigEncrypted, tags, cp.IntegrationID, cp.ResourceSetID, tenantID, cp.ID)
	if err != nil {
		return fmt.Errorf("update credential provider: %w", err)
	}
	return nil
}

func (r *CredentialProviderRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM credential_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete credential provider: %w", err)
	}
	return nil
}
