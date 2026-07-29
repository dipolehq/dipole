package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, client_workload_id, server_workload_id, tags, resource_set_id, created_at, updated_at
		 FROM access_policies WHERE tenant_id = $1 AND id = $2`, tenantID, id).
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
	search := escapeILIKE(opts.Search)
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM access_policies
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')`,
		tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count access policies: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, client_workload_id, server_workload_id, tags, resource_set_id, created_at, updated_at
		 FROM access_policies
		 WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%' OR description ILIKE '%' || $2 || '%')
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		tenantID, search, opts.PerPage, offset)
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

	err = tx.QueryRow(ctx,
		`UPDATE access_policies SET name=$1, description=$2, is_active=$3, client_workload_id=$4, server_workload_id=$5, tags=$6, resource_set_id=$7, updated_at=NOW()
		 WHERE tenant_id = $8 AND id = $9
		 RETURNING created_at, updated_at`,
		p.Name, p.Description, p.IsActive, p.ClientWorkloadID, p.ServerWorkloadID, tags, p.ResourceSetID, tenantID, p.ID).
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
	_, err := r.pool.Exec(ctx, `DELETE FROM access_policies WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete access policy: %w", err)
	}
	return nil
}

func (r *AccessPolicyRepo) FindByServerTarget(ctx context.Context, tenantID uuid.UUID, host string, port int) ([]domain.AccessPolicyWithRelations, error) {
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
		 ORDER BY ap.created_at ASC`,
		tenantID, host, port)
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

// loadRelationsForPolicies batch-loads TrustProviders, AccessConditions, and
// CredentialProviders for all matched policies via junction table JOINs.
func (r *AccessPolicyRepo) loadRelationsForPolicies(ctx context.Context, tenantID uuid.UUID, policies []domain.AccessPolicyWithRelations) error {
	ids := make([]uuid.UUID, len(policies))
	idxMap := make(map[uuid.UUID]int, len(policies))
	for i := range policies {
		ids[i] = policies[i].Policy.ID
		idxMap[policies[i].Policy.ID] = i
	}

	// Load Trust Providers (full entity rows via JOIN).
	tpRows, err := r.pool.Query(ctx,
		`SELECT ptp.policy_id, tp.id, tp.name, tp.description, tp.is_active, tp.type, tp.config, tp.match_rules, tp.edge_sdk_client_id, tp.edge_sdk_audience
		 FROM policy_trust_providers ptp
		 JOIN trust_providers tp ON tp.id = ptp.trust_provider_id AND tp.tenant_id = $1
		 WHERE ptp.policy_id = ANY($2)`,
		tenantID, ids)
	if err != nil {
		return fmt.Errorf("load trust providers for policies: %w", err)
	}
	defer tpRows.Close()
	for tpRows.Next() {
		var policyID uuid.UUID
		var tp domain.TrustProvider
		var matchRules []byte
		if err := tpRows.Scan(&policyID, &tp.ID, &tp.Name, &tp.Description, &tp.IsActive, &tp.Type, &tp.Config, &matchRules, &tp.EdgeSDKClientID, &tp.EdgeSDKAudience); err != nil {
			return fmt.Errorf("scan trust provider: %w", err)
		}
		json.Unmarshal(matchRules, &tp.MatchRules)
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].TrustProviders = append(policies[idx].TrustProviders, tp)
		}
	}
	tpRows.Close()

	// Load Access Conditions (full entity rows via JOIN).
	acRows, err := r.pool.Query(ctx,
		`SELECT pac.policy_id, ac.id, ac.name, ac.description, ac.is_active, ac.type, ac.config
		 FROM policy_access_conditions pac
		 JOIN access_conditions ac ON ac.id = pac.access_condition_id AND ac.tenant_id = $1
		 WHERE pac.policy_id = ANY($2)`,
		tenantID, ids)
	if err != nil {
		return fmt.Errorf("load access conditions for policies: %w", err)
	}
	defer acRows.Close()
	for acRows.Next() {
		var policyID uuid.UUID
		var ac domain.AccessCondition
		if err := acRows.Scan(&policyID, &ac.ID, &ac.Name, &ac.Description, &ac.IsActive, &ac.Type, &ac.Config); err != nil {
			return fmt.Errorf("scan access condition: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].AccessConditions = append(policies[idx].AccessConditions, ac)
		}
	}
	acRows.Close()

	// Load Credential Providers (full entity rows via JOIN).
	cpRows, err := r.pool.Query(ctx,
		`SELECT pcp.policy_id, cp.id, cp.name, cp.description, cp.is_active, cp.type, cp.config_encrypted, cp.integration_id
		 FROM policy_credential_providers pcp
		 JOIN credential_providers cp ON cp.id = pcp.credential_provider_id AND cp.tenant_id = $1
		 WHERE pcp.policy_id = ANY($2)`,
		tenantID, ids)
	if err != nil {
		return fmt.Errorf("load credential providers for policies: %w", err)
	}
	defer cpRows.Close()
	for cpRows.Next() {
		var policyID uuid.UUID
		var cp domain.CredentialProvider
		if err := cpRows.Scan(&policyID, &cp.ID, &cp.Name, &cp.Description, &cp.IsActive, &cp.Type, &cp.ConfigEncrypted, &cp.IntegrationID); err != nil {
			return fmt.Errorf("scan credential provider: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].CredentialProviders = append(policies[idx].CredentialProviders, cp)
		}
	}

	return nil
}

func (r *AccessPolicyRepo) syncJunctionsWithTx(ctx context.Context, tx Pool, p *domain.AccessPolicy) error {
	tx.Exec(ctx, `DELETE FROM policy_trust_providers WHERE policy_id = $1`, p.ID)
	tx.Exec(ctx, `DELETE FROM policy_credential_providers WHERE policy_id = $1`, p.ID)
	tx.Exec(ctx, `DELETE FROM policy_access_conditions WHERE policy_id = $1`, p.ID)

	for _, tpID := range p.TrustProviderIDs {
		_, err := tx.Exec(ctx, `INSERT INTO policy_trust_providers (policy_id, trust_provider_id) VALUES ($1, $2)`, p.ID, tpID)
		if err != nil {
			return fmt.Errorf("insert policy trust provider: %w", err)
		}
	}

	for _, pcp := range p.CredentialProviderIDs {
		_, err := tx.Exec(ctx, `INSERT INTO policy_credential_providers (policy_id, credential_provider_id, mapping_type, mapping_value) VALUES ($1, $2, $3, $4)`,
			p.ID, pcp.CredentialProviderID, pcp.MappingType, pcp.MappingValue)
		if err != nil {
			return fmt.Errorf("insert policy credential provider: %w", err)
		}
	}

	for _, acID := range p.AccessConditionIDs {
		_, err := tx.Exec(ctx, `INSERT INTO policy_access_conditions (policy_id, access_condition_id) VALUES ($1, $2)`, p.ID, acID)
		if err != nil {
			return fmt.Errorf("insert policy access condition: %w", err)
		}
	}
	return nil
}

func (r *AccessPolicyRepo) loadJunctions(ctx context.Context, p *domain.AccessPolicy) error {
	tpRows, err := r.pool.Query(ctx, `SELECT trust_provider_id FROM policy_trust_providers WHERE policy_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("load trust providers: %w", err)
	}
	defer tpRows.Close()
	for tpRows.Next() {
		var id uuid.UUID
		tpRows.Scan(&id)
		p.TrustProviderIDs = append(p.TrustProviderIDs, id)
	}

	cpRows, err := r.pool.Query(ctx, `SELECT credential_provider_id, mapping_type, mapping_value FROM policy_credential_providers WHERE policy_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("load credential providers: %w", err)
	}
	defer cpRows.Close()
	for cpRows.Next() {
		var pcp domain.PolicyCredentialProvider
		cpRows.Scan(&pcp.CredentialProviderID, &pcp.MappingType, &pcp.MappingValue)
		p.CredentialProviderIDs = append(p.CredentialProviderIDs, pcp)
	}

	acRows, err := r.pool.Query(ctx, `SELECT access_condition_id FROM policy_access_conditions WHERE policy_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("load access conditions: %w", err)
	}
	defer acRows.Close()
	for acRows.Next() {
		var id uuid.UUID
		acRows.Scan(&id)
		p.AccessConditionIDs = append(p.AccessConditionIDs, id)
	}

	return nil
}

func (r *AccessPolicyRepo) batchLoadJunctions(ctx context.Context, policies []domain.AccessPolicy) error {
	ids := make([]uuid.UUID, len(policies))
	idxMap := make(map[uuid.UUID]int, len(policies))
	for i := range policies {
		ids[i] = policies[i].ID
		idxMap[policies[i].ID] = i
	}

	// Trust Providers
	tpRows, err := r.pool.Query(ctx, `SELECT policy_id, trust_provider_id FROM policy_trust_providers WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load trust providers: %w", err)
	}
	defer tpRows.Close()
	for tpRows.Next() {
		var policyID, tpID uuid.UUID
		tpRows.Scan(&policyID, &tpID)
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].TrustProviderIDs = append(policies[idx].TrustProviderIDs, tpID)
		}
	}

	// Credential Providers
	cpRows, err := r.pool.Query(ctx, `SELECT policy_id, credential_provider_id, mapping_type, mapping_value FROM policy_credential_providers WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load credential providers: %w", err)
	}
	defer cpRows.Close()
	for cpRows.Next() {
		var policyID uuid.UUID
		var pcp domain.PolicyCredentialProvider
		cpRows.Scan(&policyID, &pcp.CredentialProviderID, &pcp.MappingType, &pcp.MappingValue)
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].CredentialProviderIDs = append(policies[idx].CredentialProviderIDs, pcp)
		}
	}

	// Access Conditions
	acRows, err := r.pool.Query(ctx, `SELECT policy_id, access_condition_id FROM policy_access_conditions WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load access conditions: %w", err)
	}
	defer acRows.Close()
	for acRows.Next() {
		var policyID, acID uuid.UUID
		acRows.Scan(&policyID, &acID)
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].AccessConditionIDs = append(policies[idx].AccessConditionIDs, acID)
		}
	}

	return nil
}

// wildcard matching helper (unused in SQL but kept for application-level fallback)
func matchHost(pattern, target string) bool {
	if pattern == target {
		return true
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:]
		return strings.HasSuffix(target, suffix)
	}
	return false
}

// escapeILIKE escapes special ILIKE characters (%, _, \) in a search term.
func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
