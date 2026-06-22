package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type ServerWorkloadRepo struct {
	pool Pool
}

func (r *ServerWorkloadRepo) Create(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	sw.ID = uuid.New()
	sw.TenantID = tenantID

	authConfig, _ := json.Marshal(sw.AuthConfig)
	tags, _ := json.Marshal(sw.Tags)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO server_workloads (id, tenant_id, name, description, host, port, app_protocol, tls, auth_method, auth_scheme, auth_config, url_path, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		sw.ID, tenantID, sw.Name, sw.Description, sw.Host, sw.Port, sw.AppProtocol, sw.TLS, sw.AuthMethod, sw.AuthScheme, authConfig, sw.URLPath, tags, sw.ResourceSetID)
	if err != nil {
		return fmt.Errorf("create server workload: %w", err)
	}
	return nil
}

func (r *ServerWorkloadRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ServerWorkload, error) {
	sw := &domain.ServerWorkload{}
	var authConfig, tags []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, host, port, app_protocol, tls, auth_method, auth_scheme, auth_config, url_path, tags, resource_set_id, created_at, updated_at
		 FROM server_workloads WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&sw.ID, &sw.TenantID, &sw.Name, &sw.Description, &sw.Host, &sw.Port, &sw.AppProtocol, &sw.TLS, &sw.AuthMethod, &sw.AuthScheme, &authConfig, &sw.URLPath, &tags, &sw.ResourceSetID, &sw.CreatedAt, &sw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get server workload: %w", err)
	}

	json.Unmarshal(authConfig, &sw.AuthConfig)
	json.Unmarshal(tags, &sw.Tags)
	return sw, nil
}

func (r *ServerWorkloadRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ServerWorkload, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM server_workloads WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count server workloads: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, host, port, app_protocol, tls, auth_method, auth_scheme, auth_config, url_path, tags, resource_set_id, created_at, updated_at
		 FROM server_workloads WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list server workloads: %w", err)
	}
	defer rows.Close()

	var results []domain.ServerWorkload
	for rows.Next() {
		var sw domain.ServerWorkload
		var authConfig, tags []byte
		if err := rows.Scan(&sw.ID, &sw.TenantID, &sw.Name, &sw.Description, &sw.Host, &sw.Port, &sw.AppProtocol, &sw.TLS, &sw.AuthMethod, &sw.AuthScheme, &authConfig, &sw.URLPath, &tags, &sw.ResourceSetID, &sw.CreatedAt, &sw.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan server workload: %w", err)
		}
		json.Unmarshal(authConfig, &sw.AuthConfig)
		json.Unmarshal(tags, &sw.Tags)
		results = append(results, sw)
	}
	return results, total, nil
}

func (r *ServerWorkloadRepo) Update(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error {
	authConfig, _ := json.Marshal(sw.AuthConfig)
	tags, _ := json.Marshal(sw.Tags)

	_, err := r.pool.Exec(ctx,
		`UPDATE server_workloads SET name=$1, description=$2, host=$3, port=$4, app_protocol=$5, tls=$6, auth_method=$7, auth_scheme=$8, auth_config=$9, url_path=$10, tags=$11, resource_set_id=$12, updated_at=NOW()
		 WHERE tenant_id = $13 AND id = $14`,
		sw.Name, sw.Description, sw.Host, sw.Port, sw.AppProtocol, sw.TLS, sw.AuthMethod, sw.AuthScheme, authConfig, sw.URLPath, tags, sw.ResourceSetID, tenantID, sw.ID)
	if err != nil {
		return fmt.Errorf("update server workload: %w", err)
	}
	return nil
}

func (r *ServerWorkloadRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM server_workloads WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete server workload: %w", err)
	}
	return nil
}
