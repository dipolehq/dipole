package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type ResourceSetRepo struct {
	pool Pool
}

func (r *ResourceSetRepo) Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	rs.ID = uuid.New()
	rs.TenantID = tenantID

	_, err := r.pool.Exec(ctx,
		`INSERT INTO resource_sets (id, tenant_id, name, description) VALUES ($1, $2, $3, $4)`,
		rs.ID, tenantID, rs.Name, rs.Description)
	if err != nil {
		return fmt.Errorf("create resource set: %w", err)
	}
	return nil
}

func (r *ResourceSetRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ResourceSet, error) {
	rs := &domain.ResourceSet{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, created_at, updated_at FROM resource_sets WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&rs.ID, &rs.TenantID, &rs.Name, &rs.Description, &rs.CreatedAt, &rs.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get resource set: %w", err)
	}
	return rs, nil
}

func (r *ResourceSetRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM resource_sets WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count resource sets: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, created_at, updated_at FROM resource_sets
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list resource sets: %w", err)
	}
	defer rows.Close()

	var results []domain.ResourceSet
	for rows.Next() {
		var rs domain.ResourceSet
		if err := rows.Scan(&rs.ID, &rs.TenantID, &rs.Name, &rs.Description, &rs.CreatedAt, &rs.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan resource set: %w", err)
		}
		results = append(results, rs)
	}
	return results, total, nil
}

func (r *ResourceSetRepo) Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE resource_sets SET name=$1, description=$2, updated_at=NOW() WHERE tenant_id = $3 AND id = $4`,
		rs.Name, rs.Description, tenantID, rs.ID)
	if err != nil {
		return fmt.Errorf("update resource set: %w", err)
	}
	return nil
}

func (r *ResourceSetRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM resource_sets WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete resource set: %w", err)
	}
	return nil
}

func (r *ResourceSetRepo) GetRoleIDs(ctx context.Context, tenantID uuid.UUID, resourceSetID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT role_id FROM role_resource_set_assignments WHERE resource_set_id = $1`, resourceSetID)
	if err != nil {
		return nil, fmt.Errorf("get resource set role assignments: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan role id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
