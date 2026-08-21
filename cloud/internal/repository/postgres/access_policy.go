// access_policy.go implements the AccessPolicyRepository for PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AccessPolicyRepo struct {
	pool TxPool
}

func (r *AccessPolicyRepo) Create(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	p.ID = uuid.New()
	p.TenantID = tenantID
	tags, _ := json.Marshal(p.Tags)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO access_policies (id, tenant_id, name, description, is_active, client_workload_id, server_workload_id, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING created_at, updated_at`,
		p.ID, tenantID, p.Name, p.Description, p.IsActive, p.ClientWorkloadID, p.ServerWorkloadID, tags, p.ResourceSetID).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create access policy: %w", err)
	}

	if err := r.syncJunctionsWithTx(ctx, tx, p); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AccessPolicyRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AccessPolicy, error) {
	p := &domain.AccessPolicy{}
	var tags []byte

	args := []interface{}{tenantID, id}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, client_workload_id, server_workload_id, tags, resource_set_id, created_at, updated_at
		 FROM access_policies WHERE tenant_id = $1 AND id = $2`+rsFilter, args...).
		Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsActive, &p.ClientWorkloadID, &p.ServerWorkloadID, &tags, &p.ResourceSetID, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get access policy: %w", err)
	}

	json.Unmarshal(tags, &p.Tags)
	if err := r.loadJunctions(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *AccessPolicyRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessPolicy, int, error) {
	where, countArgs, n := buildTenantSearchFilter(tenantID, opts.Search)
	rsFilter := ResourceSetFilter(ctx, &countArgs, &n)

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM access_policies WHERE `+where+rsFilter,
		countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count access policies: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	where, listArgs, n := buildTenantSearchFilter(tenantID, opts.Search)
	rsFilter = ResourceSetFilter(ctx, &listArgs, &n)
	limitOffset := appendLimitOffset(&listArgs, &n, opts.PerPage, offset)

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, client_workload_id, server_workload_id, tags, resource_set_id, created_at, updated_at
		 FROM access_policies
		 WHERE `+where+rsFilter+`
		 ORDER BY created_at DESC `+limitOffset,
		listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list access policies: %w", err)
	}
	defer rows.Close()

	var results []domain.AccessPolicy
	for rows.Next() {
		var p domain.AccessPolicy
		var tags []byte
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Description, &p.IsActive, &p.ClientWorkloadID, &p.ServerWorkloadID, &tags, &p.ResourceSetID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan access policy: %w", err)
		}
		json.Unmarshal(tags, &p.Tags)
		results = append(results, p)
	}

	if len(results) > 0 {
		if err := r.batchLoadJunctions(ctx, results); err != nil {
			return nil, 0, err
		}
	}
	return results, total, nil
}

func (r *AccessPolicyRepo) Update(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error {
	tags, _ := json.Marshal(p.Tags)

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	args := []interface{}{p.Name, p.Description, p.IsActive, p.ClientWorkloadID, p.ServerWorkloadID, tags, p.ResourceSetID, tenantID, p.ID}
	n := 9
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	err = tx.QueryRow(ctx,
		`UPDATE access_policies SET name=$1, description=$2, is_active=$3, client_workload_id=$4, server_workload_id=$5, tags=$6, resource_set_id=$7, updated_at=NOW()
		 WHERE tenant_id = $8 AND id = $9`+rsFilter+`
		 RETURNING created_at, updated_at`,
		args...).
		Scan(&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update access policy: %w", err)
	}

	if err := r.syncJunctionsWithTx(ctx, tx, p); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *AccessPolicyRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	args := []interface{}{tenantID, id}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	_, err := r.pool.Exec(ctx, `DELETE FROM access_policies WHERE tenant_id = $1 AND id = $2`+rsFilter, args...)
	if err != nil {
		return fmt.Errorf("delete access policy: %w", err)
	}
	return nil
}

func (r *AccessPolicyRepo) FindByServerTarget(ctx context.Context, tenantID uuid.UUID, host string, port int, resourceSetID *uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	defaultRS := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	rsFilter := defaultRS
	if resourceSetID != nil {
		rsFilter = *resourceSetID
	}

	rows, err := r.pool.Query(ctx,
		`SELECT ap.id, ap.tenant_id, ap.name, ap.description, ap.is_active, ap.client_workload_id, ap.server_workload_id, ap.tags, ap.resource_set_id, ap.created_at, ap.updated_at,
		        cw.id, cw.name, cw.identifiers, cw.tags,
		        sw.id, sw.name, sw.host, sw.port, sw.app_protocol, sw.tls, sw.auth_method, sw.auth_scheme, sw.auth_config, sw.url_path
		 FROM access_policies ap
		 JOIN client_workloads cw ON ap.client_workload_id = cw.id
		 JOIN server_workloads sw ON ap.server_workload_id = sw.id
		 WHERE ap.tenant_id = $1 AND ap.is_active = true
		   AND (sw.host = $2 OR (sw.host LIKE '*%' AND $2 LIKE '%' || replace(replace(substring(sw.host from 2), '%', '\%'), '_', '\_')))
		   AND sw.port = $3
		   AND (ap.resource_set_id IS NULL OR ap.resource_set_id = $4 OR ap.resource_set_id = $5)
		 ORDER BY ap.created_at ASC`,
		tenantID, host, port, rsFilter, defaultRS)
	if err != nil {
		return nil, fmt.Errorf("find policies by server target: %w", err)
	}
	defer rows.Close()

	var results []domain.AccessPolicyWithRelations
	for rows.Next() {
		var rel domain.AccessPolicyWithRelations
		var pTags, cwIdentifiers, cwTags, swAuthConfig []byte

		if err := rows.Scan(
			&rel.Policy.ID, &rel.Policy.TenantID, &rel.Policy.Name, &rel.Policy.Description, &rel.Policy.IsActive, &rel.Policy.ClientWorkloadID, &rel.Policy.ServerWorkloadID, &pTags, &rel.Policy.ResourceSetID, &rel.Policy.CreatedAt, &rel.Policy.UpdatedAt,
			&rel.ClientWorkload.ID, &rel.ClientWorkload.Name, &cwIdentifiers, &cwTags,
			&rel.ServerWorkload.ID, &rel.ServerWorkload.Name, &rel.ServerWorkload.Host, &rel.ServerWorkload.Port, &rel.ServerWorkload.AppProtocol, &rel.ServerWorkload.TLS, &rel.ServerWorkload.AuthMethod, &rel.ServerWorkload.AuthScheme, &swAuthConfig, &rel.ServerWorkload.URLPath,
		); err != nil {
			return nil, fmt.Errorf("scan policy relation: %w", err)
		}

		json.Unmarshal(pTags, &rel.Policy.Tags)
		json.Unmarshal(cwIdentifiers, &rel.ClientWorkload.Identifiers)
		json.Unmarshal(cwTags, &rel.ClientWorkload.Tags)
		json.Unmarshal(swAuthConfig, &rel.ServerWorkload.AuthConfig)

		results = append(results, rel)
	}
	rows.Close()

	if len(results) > 0 {
		if err := r.loadRelationsForPolicies(ctx, tenantID, results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *AccessPolicyRepo) FindByAppProtocol(ctx context.Context, tenantID uuid.UUID, protocol string) ([]domain.AccessPolicyWithRelations, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ap.id, ap.tenant_id, ap.name, ap.description, ap.is_active, ap.client_workload_id, ap.server_workload_id, ap.tags, ap.resource_set_id, ap.created_at, ap.updated_at,
		        cw.id, cw.name, cw.identifiers, cw.tags,
		        sw.id, sw.name, sw.host, sw.port, sw.app_protocol, sw.tls, sw.auth_method, sw.auth_scheme, sw.auth_config, sw.url_path
		 FROM access_policies ap
		 JOIN client_workloads cw ON ap.client_workload_id = cw.id
		 JOIN server_workloads sw ON ap.server_workload_id = sw.id
		 WHERE ap.tenant_id = $1 AND ap.is_active = true AND sw.app_protocol = $2
		 ORDER BY ap.created_at ASC`,
		tenantID, protocol)
	if err != nil {
		return nil, fmt.Errorf("find policies by app protocol: %w", err)
	}
	defer rows.Close()

	var results []domain.AccessPolicyWithRelations
	for rows.Next() {
		var rel domain.AccessPolicyWithRelations
		var pTags, cwIdentifiers, cwTags, swAuthConfig []byte

		if err := rows.Scan(
			&rel.Policy.ID, &rel.Policy.TenantID, &rel.Policy.Name, &rel.Policy.Description, &rel.Policy.IsActive, &rel.Policy.ClientWorkloadID, &rel.Policy.ServerWorkloadID, &pTags, &rel.Policy.ResourceSetID, &rel.Policy.CreatedAt, &rel.Policy.UpdatedAt,
			&rel.ClientWorkload.ID, &rel.ClientWorkload.Name, &cwIdentifiers, &cwTags,
			&rel.ServerWorkload.ID, &rel.ServerWorkload.Name, &rel.ServerWorkload.Host, &rel.ServerWorkload.Port, &rel.ServerWorkload.AppProtocol, &rel.ServerWorkload.TLS, &rel.ServerWorkload.AuthMethod, &rel.ServerWorkload.AuthScheme, &swAuthConfig, &rel.ServerWorkload.URLPath,
		); err != nil {
			return nil, fmt.Errorf("scan policy relation: %w", err)
		}

		json.Unmarshal(pTags, &rel.Policy.Tags)
		json.Unmarshal(cwIdentifiers, &rel.ClientWorkload.Identifiers)
		json.Unmarshal(cwTags, &rel.ClientWorkload.Tags)
		json.Unmarshal(swAuthConfig, &rel.ServerWorkload.AuthConfig)

		results = append(results, rel)
	}
	rows.Close()

	if len(results) > 0 {
		if err := r.loadRelationsForPolicies(ctx, tenantID, results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *AccessPolicyRepo) ListActiveWithRelations(ctx context.Context, tenantID uuid.UUID) ([]domain.AccessPolicyWithRelations, error) {
	args := []interface{}{tenantID}
	n := 1
	rsFilter := ResourceSetFilter(ctx, &args, &n)
	apRsFilter := ""
	if rsFilter != "" {
		apRsFilter = fmt.Sprintf(" AND ap.resource_set_id = $%d", n)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT ap.id, ap.tenant_id, ap.name, ap.description, ap.is_active, ap.client_workload_id, ap.server_workload_id, ap.tags, ap.resource_set_id, ap.created_at, ap.updated_at,
		        cw.id, cw.name, cw.identifiers, cw.tags,
		        sw.id, sw.name, sw.host, sw.port, sw.app_protocol, sw.tls, sw.auth_method, sw.auth_scheme, sw.auth_config, sw.url_path
		 FROM access_policies ap
		 JOIN client_workloads cw ON ap.client_workload_id = cw.id
		 JOIN server_workloads sw ON ap.server_workload_id = sw.id
		 WHERE ap.tenant_id = $1 AND ap.is_active = true`+apRsFilter+`
		 ORDER BY ap.created_at ASC`,
		args...)
	if err != nil {
		return nil, fmt.Errorf("list active policies with relations: %w", err)
	}
	defer rows.Close()

	var results []domain.AccessPolicyWithRelations
	for rows.Next() {
		var rel domain.AccessPolicyWithRelations
		var pTags, cwIdentifiers, cwTags, swAuthConfig []byte

		if err := rows.Scan(
			&rel.Policy.ID, &rel.Policy.TenantID, &rel.Policy.Name, &rel.Policy.Description, &rel.Policy.IsActive, &rel.Policy.ClientWorkloadID, &rel.Policy.ServerWorkloadID, &pTags, &rel.Policy.ResourceSetID, &rel.Policy.CreatedAt, &rel.Policy.UpdatedAt,
			&rel.ClientWorkload.ID, &rel.ClientWorkload.Name, &cwIdentifiers, &cwTags,
			&rel.ServerWorkload.ID, &rel.ServerWorkload.Name, &rel.ServerWorkload.Host, &rel.ServerWorkload.Port, &rel.ServerWorkload.AppProtocol, &rel.ServerWorkload.TLS, &rel.ServerWorkload.AuthMethod, &rel.ServerWorkload.AuthScheme, &swAuthConfig, &rel.ServerWorkload.URLPath,
		); err != nil {
			return nil, fmt.Errorf("scan policy relation: %w", err)
		}

		json.Unmarshal(pTags, &rel.Policy.Tags)
		json.Unmarshal(cwIdentifiers, &rel.ClientWorkload.Identifiers)
		json.Unmarshal(cwTags, &rel.ClientWorkload.Tags)
		json.Unmarshal(swAuthConfig, &rel.ServerWorkload.AuthConfig)

		results = append(results, rel)
	}
	rows.Close()

	if len(results) > 0 {
		if err := r.loadRelationsForPolicies(ctx, tenantID, results); err != nil {
			return nil, err
		}
	}

	return results, nil
}

func (r *AccessPolicyRepo) CountInactive(ctx context.Context, tenantID uuid.UUID) int {
	var count int
	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM access_policies WHERE tenant_id = $1 AND is_active = false`,
		tenantID).Scan(&count)
	return count
}

func (r *AccessPolicyRepo) IsHostnameAuthorizedForTLS(ctx context.Context, tenantID uuid.UUID, hostname string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM access_policies ap
			JOIN server_workloads sw ON sw.id = ap.server_workload_id AND sw.tenant_id = $1
			WHERE ap.tenant_id = $1
			  AND ap.is_active = true
			  AND sw.host = $2
			  AND sw.tls = true
		)`,
		tenantID, hostname).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check hostname authorization: %w", err)
	}
	return exists, nil
}
