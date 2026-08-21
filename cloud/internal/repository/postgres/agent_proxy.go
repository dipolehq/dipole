package postgres

import (
	"context"
	"fmt"
	"time"

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

func (r *AgentProxyRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AgentProxy, int, error) {
	where, countArgs, n := buildTenantFilter(tenantID)
	rsFilter := ResourceSetFilter(ctx, &countArgs, &n)

	var total int
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agent_proxies WHERE `+where+rsFilter, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count agent proxies: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	where, listArgs, n := buildTenantFilter(tenantID)
	rsFilter = ResourceSetFilter(ctx, &listArgs, &n)
	limitOffset := appendLimitOffset(&listArgs, &n, opts.PerPage, offset)

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, agent_controller_id, status, last_health_at, resource_set_id, registered_at
		 FROM agent_proxies WHERE `+where+rsFilter+`
		 ORDER BY registered_at DESC NULLS LAST
		 `+limitOffset,
		listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list agent proxies: %w", err)
	}
	defer rows.Close()

	var results []domain.AgentProxy
	for rows.Next() {
		var p domain.AgentProxy
		if err := rows.Scan(&p.ID, &p.TenantID, &p.AgentControllerID, &p.Status, &p.LastHealthAt, &p.ResourceSetID, &p.RegisteredAt); err != nil {
			return nil, 0, fmt.Errorf("scan agent proxy: %w", err)
		}
		results = append(results, p)
	}
	return results, total, nil
}

func (r *AgentProxyRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentProxy, error) {
	proxy := &domain.AgentProxy{}

	args := []interface{}{tenantID, id}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, agent_controller_id, status, last_health_at, resource_set_id, registered_at
		 FROM agent_proxies WHERE tenant_id = $1 AND id = $2`+rsFilter, args...).
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

func (r *AgentProxyRepo) SetSessionTokenHash(ctx context.Context, tenantID uuid.UUID, proxyID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE agent_proxies SET session_token_hash = $3, session_token_expires_at = $4 WHERE tenant_id = $1 AND id = $2`,
		tenantID, proxyID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("set agent proxy session token hash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("set agent proxy session token hash: proxy not found")
	}
	return nil
}

func (r *AgentProxyRepo) GetBySessionTokenHash(ctx context.Context, tokenHash string) (*domain.AgentProxy, error) {
	proxy := &domain.AgentProxy{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, agent_controller_id, status, last_health_at, resource_set_id, registered_at, session_token_expires_at
		 FROM agent_proxies WHERE session_token_hash = $1 AND session_token_expires_at > now()`, tokenHash).
		Scan(&proxy.ID, &proxy.TenantID, &proxy.AgentControllerID, &proxy.Status, &proxy.LastHealthAt, &proxy.ResourceSetID, &proxy.RegisteredAt, &proxy.SessionTokenExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent proxy by session token hash: %w", err)
	}
	return proxy, nil
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
