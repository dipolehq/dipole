package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type ClientWorkloadRepo struct {
	pool Pool
}

func (r *ClientWorkloadRepo) Create(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	cw.ID = uuid.New()
	cw.TenantID = tenantID

	identifiers, _ := json.Marshal(cw.Identifiers)
	tags, _ := json.Marshal(cw.Tags)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO client_workloads (id, tenant_id, name, description, identifiers, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cw.ID, tenantID, cw.Name, cw.Description, identifiers, tags, cw.ResourceSetID)
	if err != nil {
		return fmt.Errorf("create client workload: %w", err)
	}
	return nil
}

func (r *ClientWorkloadRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ClientWorkload, error) {
	cw := &domain.ClientWorkload{}
	var identifiers, tags []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, identifiers, tags, resource_set_id, created_at, updated_at
		 FROM client_workloads WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&cw.ID, &cw.TenantID, &cw.Name, &cw.Description, &identifiers, &tags, &cw.ResourceSetID, &cw.CreatedAt, &cw.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get client workload: %w", err)
	}

	json.Unmarshal(identifiers, &cw.Identifiers)
	json.Unmarshal(tags, &cw.Tags)
	return cw, nil
}

func (r *ClientWorkloadRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ClientWorkload, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM client_workloads WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count client workloads: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, identifiers, tags, resource_set_id, created_at, updated_at
		 FROM client_workloads WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list client workloads: %w", err)
	}
	defer rows.Close()

	var results []domain.ClientWorkload
	for rows.Next() {
		var cw domain.ClientWorkload
		var identifiers, tags []byte
		if err := rows.Scan(&cw.ID, &cw.TenantID, &cw.Name, &cw.Description, &identifiers, &tags, &cw.ResourceSetID, &cw.CreatedAt, &cw.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan client workload: %w", err)
		}
		json.Unmarshal(identifiers, &cw.Identifiers)
		json.Unmarshal(tags, &cw.Tags)
		results = append(results, cw)
	}
	return results, total, nil
}

func (r *ClientWorkloadRepo) Update(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	identifiers, _ := json.Marshal(cw.Identifiers)
	tags, _ := json.Marshal(cw.Tags)

	_, err := r.pool.Exec(ctx,
		`UPDATE client_workloads SET name = $1, description = $2, identifiers = $3, tags = $4, resource_set_id = $5, updated_at = NOW()
		 WHERE tenant_id = $6 AND id = $7`,
		cw.Name, cw.Description, identifiers, tags, cw.ResourceSetID, tenantID, cw.ID)
	if err != nil {
		return fmt.Errorf("update client workload: %w", err)
	}
	return nil
}

func (r *ClientWorkloadRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM client_workloads WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete client workload: %w", err)
	}
	return nil
}
