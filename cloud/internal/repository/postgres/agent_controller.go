package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AgentControllerRepo struct {
	pool Pool
}

func (r *AgentControllerRepo) Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AgentController) error {
	ac.ID = uuid.New()
	ac.TenantID = tenantID
	tags, _ := json.Marshal(ac.Tags)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO agent_controllers (id, tenant_id, name, description, is_active, status, device_code, device_code_expires_at, trust_provider_id, tls_hostname, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		 RETURNING created_at, updated_at`,
		ac.ID, tenantID, ac.Name, ac.Description, ac.IsActive, ac.Status, ac.DeviceCode, ac.DeviceCodeExpiresAt, ac.TrustProviderID, ac.TLSHostname, tags, ac.ResourceSetID).
		Scan(&ac.CreatedAt, &ac.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent controller: %w", err)
	}
	return nil
}

func (r *AgentControllerRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentController, error) {
	ac := &domain.AgentController{}
	var tags []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, status, device_code, device_code_expires_at, trust_provider_id, tls_hostname, tags, last_health_at, resource_set_id, created_at, updated_at
		 FROM agent_controllers WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.TLSHostname, &tags, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt, &ac.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent controller: %w", err)
	}
	json.Unmarshal(tags, &ac.Tags)
	return ac, nil
}

func (r *AgentControllerRepo) GetByDeviceCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.AgentController, error) {
	ac := &domain.AgentController{}
	var tags []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, status, device_code, device_code_expires_at, trust_provider_id, tls_hostname, tags, last_health_at, resource_set_id, created_at, updated_at
		 FROM agent_controllers WHERE tenant_id = $1 AND device_code = $2`, tenantID, code).
		Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.TLSHostname, &tags, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt, &ac.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent controller by device code: %w", err)
	}
	json.Unmarshal(tags, &ac.Tags)
	return ac, nil
}

func (r *AgentControllerRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AgentController, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM agent_controllers WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agent controllers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, status, device_code, device_code_expires_at, trust_provider_id, tls_hostname, tags, last_health_at, resource_set_id, created_at, updated_at
		 FROM agent_controllers
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent controllers: %w", err)
	}
	defer rows.Close()

	var results []domain.AgentController
	for rows.Next() {
		var ac domain.AgentController
		var tags []byte
		if err := rows.Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.TLSHostname, &tags, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt, &ac.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent controller: %w", err)
		}
		json.Unmarshal(tags, &ac.Tags)
		results = append(results, ac)
	}
	return results, total, nil
}

func (r *AgentControllerRepo) Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AgentController) error {
	tags, _ := json.Marshal(ac.Tags)
	_, err := r.pool.Exec(ctx,
		`UPDATE agent_controllers SET name=$1, description=$2, is_active=$3, status=$4, device_code=$5, device_code_expires_at=$6, trust_provider_id=$7, tls_hostname=$8, tags=$9, last_health_at=$10, resource_set_id=$11, updated_at=NOW()
		 WHERE tenant_id = $12 AND id = $13`,
		ac.Name, ac.Description, ac.IsActive, ac.Status, ac.DeviceCode, ac.DeviceCodeExpiresAt, ac.TrustProviderID, ac.TLSHostname, tags, ac.LastHealthAt, ac.ResourceSetID, tenantID, ac.ID)
	if err != nil {
		return fmt.Errorf("update agent controller: %w", err)
	}
	return nil
}

func (r *AgentControllerRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM agent_controllers WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete agent controller: %w", err)
	}
	return nil
}
