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
	// Respect a caller-supplied ID (signup.go creates the tenant's Default
	// resource set with the well-known scope.DefaultResourceSetID so it
	// matches every resource-scoped table's column default) and only
	// generate one when the caller hasn't set one — the same pattern used
	// by every other repository's Create (e.g. TrustProviderRepo.Create).
	if rs.ID == uuid.Nil {
		rs.ID = uuid.New()
	}
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

// GetRoleIDs is tenant-scoped (matching RoleRepo.AssignResourceSets/
// GetResourceSetIDs) so a lookup can never cross a tenant boundary even
// if a resource_set_id collided with another tenant's.
func (r *ResourceSetRepo) GetRoleIDs(ctx context.Context, tenantID uuid.UUID, resourceSetID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT role_id FROM role_resource_set_assignments WHERE tenant_id = $1 AND resource_set_id = $2`,
		tenantID, resourceSetID)
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

// GetResourceCounts returns per-resource-type counts for the given resource set IDs.
// Uses a single query with UNION ALL for efficiency — one database roundtrip regardless
// of how many resource sets are requested.
func (r *ResourceSetRepo) GetResourceCounts(ctx context.Context, tenantID uuid.UUID, resourceSetIDs []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error) {
	if len(resourceSetIDs) == 0 {
		return map[uuid.UUID]domain.ResourceSetCounts{}, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT resource_set_id, 'access_policies' AS resource_type, COUNT(*) AS cnt
		FROM access_policies WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
		UNION ALL
		SELECT resource_set_id, 'client_workloads', COUNT(*)
		FROM client_workloads WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
		UNION ALL
		SELECT resource_set_id, 'trust_providers', COUNT(*)
		FROM trust_providers WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
		UNION ALL
		SELECT resource_set_id, 'access_conditions', COUNT(*)
		FROM access_conditions WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
		UNION ALL
		SELECT resource_set_id, 'credential_providers', COUNT(*)
		FROM credential_providers WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
		UNION ALL
		SELECT resource_set_id, 'server_workloads', COUNT(*)
		FROM server_workloads WHERE tenant_id = $1 AND resource_set_id = ANY($2)
		GROUP BY resource_set_id
	`, tenantID, resourceSetIDs)
	if err != nil {
		return nil, fmt.Errorf("get resource counts: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]domain.ResourceSetCounts, len(resourceSetIDs))
	for rows.Next() {
		var rsID uuid.UUID
		var resourceType string
		var count int
		if err := rows.Scan(&rsID, &resourceType, &count); err != nil {
			return nil, fmt.Errorf("scan resource count: %w", err)
		}
		c := result[rsID]
		switch resourceType {
		case "access_policies":
			c.AccessPolicies = count
		case "client_workloads":
			c.ClientWorkloads = count
		case "trust_providers":
			c.TrustProviders = count
		case "access_conditions":
			c.AccessConditions = count
		case "credential_providers":
			c.CredentialProviders = count
		case "server_workloads":
			c.ServerWorkloads = count
		}
		result[rsID] = c
	}
	return result, nil
}
