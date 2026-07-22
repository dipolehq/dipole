package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AgentProxyRepo struct {
	pool Pool
}

func (r *AgentProxyRepo) Create(ctx context.Context, tenantID uuid.UUID, proxy *domain.AgentProxy) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO agent_proxies (id, tenant_id, agent_controller_id, status, resource_set_id, registered_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		proxy.ID, tenantID, proxy.AgentControllerID, proxy.Status, proxy.ResourceSetID, proxy.RegisteredAt)
	if err != nil {
		return fmt.Errorf("create agent proxy: %w", err)
	}
	return nil
}

func (r *AgentProxyRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentProxy, error) {
	proxy := &domain.AgentProxy{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, agent_controller_id, status, last_health_at, resource_set_id, registered_at
		 FROM agent_proxies WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&proxy.ID, &proxy.TenantID, &proxy.AgentControllerID, &proxy.Status, &proxy.LastHealthAt, &proxy.ResourceSetID, &proxy.RegisteredAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent proxy: %w", err)
	}
	return proxy, nil
}

func (r *AgentProxyRepo) UpdateHealth(ctx context.Context, tenantID uuid.UUID, proxyID uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE agent_proxies SET status = $3, last_health_at = NOW() WHERE tenant_id = $1 AND id = $2`,
		tenantID, proxyID, status)
	if err != nil {
		return fmt.Errorf("update agent proxy health: %w", err)
	}
	return nil
}

func (r *AgentProxyRepo) ListByController(ctx context.Context, tenantID uuid.UUID, controllerID uuid.UUID) ([]domain.AgentProxy, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, agent_controller_id, status, last_health_at, resource_set_id, registered_at
		 FROM agent_proxies WHERE tenant_id = $1 AND agent_controller_id = $2
		 ORDER BY registered_at DESC`, tenantID, controllerID)
	if err != nil {
		return nil, fmt.Errorf("list agent proxies by controller: %w", err)
	}
	defer rows.Close()

	var results []domain.AgentProxy
	for rows.Next() {
		var proxy domain.AgentProxy
		if err := rows.Scan(&proxy.ID, &proxy.TenantID, &proxy.AgentControllerID, &proxy.Status, &proxy.LastHealthAt, &proxy.ResourceSetID, &proxy.RegisteredAt); err != nil {
			return nil, fmt.Errorf("scan agent proxy: %w", err)
		}
		results = append(results, proxy)
	}
	return results, nil
}
