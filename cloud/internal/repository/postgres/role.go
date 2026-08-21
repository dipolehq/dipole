// role.go implements the RoleRepository for PostgreSQL.
package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type RoleRepo struct {
	pool Pool
}

func (r *RoleRepo) Create(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error {
	role.ID = uuid.New()
	role.TenantID = tenantID
	_, err := r.pool.Exec(ctx,
		`INSERT INTO roles (id, tenant_id, name, description, is_active, permissions) VALUES ($1, $2, $3, $4, $5, $6)`,
		role.ID, tenantID, role.Name, role.Description, role.IsActive, role.Permissions)
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	return nil
}

func (r *RoleRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Role, error) {
	role := &domain.Role{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, permissions, created_at, updated_at FROM roles WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsActive, &role.Permissions, &role.CreatedAt, &role.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get role: %w", err)
	}
	return role, nil
}

func (r *RoleRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.Role, int, error) {
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM roles WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count roles: %w", err)
	}
	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, permissions, created_at, updated_at
		 FROM roles WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()
	var results []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Name, &role.Description, &role.IsActive, &role.Permissions, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan role: %w", err)
		}
		results = append(results, role)
	}
	return results, total, nil
}

func (r *RoleRepo) Update(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE roles SET name=$1, description=$2, is_active=$3, permissions=$4, updated_at=NOW() WHERE tenant_id = $5 AND id = $6`,
		role.Name, role.Description, role.IsActive, role.Permissions, tenantID, role.ID)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	return nil
}

func (r *RoleRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM roles WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (r *RoleRepo) AssignResourceSets(ctx context.Context, tenantID uuid.UUID, roleID uuid.UUID, resourceSetIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM role_resource_set_assignments WHERE tenant_id = $1 AND role_id = $2`,
		tenantID, roleID)
	if err != nil {
		return fmt.Errorf("clear role resource set assignments: %w", err)
	}
	for _, rsID := range resourceSetIDs {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO role_resource_set_assignments (tenant_id, role_id, resource_set_id) VALUES ($1, $2, $3)`,
			tenantID, roleID, rsID)
		if err != nil {
			return fmt.Errorf("assign resource set to role: %w", err)
		}
	}
	return nil
}

func (r *RoleRepo) GetResourceSetIDs(ctx context.Context, tenantID uuid.UUID, roleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT resource_set_id FROM role_resource_set_assignments WHERE tenant_id = $1 AND role_id = $2`,
		tenantID, roleID)
	if err != nil {
		return nil, fmt.Errorf("get role resource set assignments: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan resource set id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
