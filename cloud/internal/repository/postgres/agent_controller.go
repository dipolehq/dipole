package postgres

import (
	"context"
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

	_, err := r.pool.Exec(ctx,
		`INSERT INTO agent_controllers (id, tenant_id, name, status, device_code, device_code_expires_at, trust_provider_id, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		ac.ID, tenantID, ac.Name, ac.Status, ac.DeviceCode, ac.DeviceCodeExpiresAt, ac.TrustProviderID, ac.ResourceSetID)
	if err != nil {
		return fmt.Errorf("create agent controller: %w", err)
	}
	return nil
}

func (r *AgentControllerRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentController, error) {
	ac := &domain.AgentController{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, status, device_code, device_code_expires_at, trust_provider_id, last_health_at, resource_set_id, created_at
		 FROM agent_controllers WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent controller: %w", err)
	}
	return ac, nil
}

func (r *AgentControllerRepo) GetByDeviceCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.AgentController, error) {
	ac := &domain.AgentController{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, status, device_code, device_code_expires_at, trust_provider_id, last_health_at, resource_set_id, created_at
		 FROM agent_controllers WHERE tenant_id = $1 AND device_code = $2`, tenantID, code).
		Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent controller by device code: %w", err)
	}
	return ac, nil
}

func (r *AgentControllerRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AgentController, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_controllers WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agent controllers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, status, device_code, device_code_expires_at, trust_provider_id, last_health_at, resource_set_id, created_at
		 FROM agent_controllers WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent controllers: %w", err)
	}
	defer rows.Close()

	var results []domain.AgentController
	for rows.Next() {
		var ac domain.AgentController
		if err := rows.Scan(&ac.ID, &ac.TenantID, &ac.Name, &ac.Status, &ac.DeviceCode, &ac.DeviceCodeExpiresAt, &ac.TrustProviderID, &ac.LastHealthAt, &ac.ResourceSetID, &ac.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent controller: %w", err)
		}
		results = append(results, ac)
	}
	return results, total, nil
}

func (r *AgentControllerRepo) Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AgentController) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE agent_controllers SET name=$1, status=$2, device_code=$3, device_code_expires_at=$4, trust_provider_id=$5, last_health_at=$6, resource_set_id=$7
		 WHERE tenant_id = $8 AND id = $9`,
		ac.Name, ac.Status, ac.DeviceCode, ac.DeviceCodeExpiresAt, ac.TrustProviderID, ac.LastHealthAt, ac.ResourceSetID, tenantID, ac.ID)
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
