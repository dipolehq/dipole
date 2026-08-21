// access_policy_junctions.go manages the policy_trust_providers, policy_credential_providers, and policy_access_conditions junction tables that back AccessPolicyRepo's many-to-many relations.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func (r *AccessPolicyRepo) loadRelationsForPolicies(ctx context.Context, tenantID uuid.UUID, policies []domain.AccessPolicyWithRelations) error {
	ids := make([]uuid.UUID, len(policies))
	idxMap := make(map[uuid.UUID]int, len(policies))
	for i := range policies {
		ids[i] = policies[i].Policy.ID
		idxMap[policies[i].Policy.ID] = i
	}

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
		if err := json.Unmarshal(matchRules, &tp.MatchRules); err != nil {
			return fmt.Errorf("unmarshal trust provider match rules: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].TrustProviders = append(policies[idx].TrustProviders, tp)
		}
	}
	if err := tpRows.Err(); err != nil {
		return fmt.Errorf("iterate trust providers for policies: %w", err)
	}

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
	if err := acRows.Err(); err != nil {
		return fmt.Errorf("iterate access conditions for policies: %w", err)
	}

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
	if err := cpRows.Err(); err != nil {
		return fmt.Errorf("iterate credential providers for policies: %w", err)
	}

	return nil
}

func (r *AccessPolicyRepo) syncJunctionsWithTx(ctx context.Context, tx Pool, p *domain.AccessPolicy) error {
	if _, err := tx.Exec(ctx, `DELETE FROM policy_trust_providers WHERE policy_id = $1`, p.ID); err != nil {
		return fmt.Errorf("clear policy trust providers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM policy_credential_providers WHERE policy_id = $1`, p.ID); err != nil {
		return fmt.Errorf("clear policy credential providers: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM policy_access_conditions WHERE policy_id = $1`, p.ID); err != nil {
		return fmt.Errorf("clear policy access conditions: %w", err)
	}

	for _, tpID := range p.TrustProviderIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO policy_trust_providers (policy_id, trust_provider_id) VALUES ($1, $2)`, p.ID, tpID); err != nil {
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
		if _, err := tx.Exec(ctx, `INSERT INTO policy_access_conditions (policy_id, access_condition_id) VALUES ($1, $2)`, p.ID, acID); err != nil {
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
		if err := tpRows.Scan(&id); err != nil {
			return fmt.Errorf("scan trust provider id: %w", err)
		}
		p.TrustProviderIDs = append(p.TrustProviderIDs, id)
	}
	if err := tpRows.Err(); err != nil {
		return fmt.Errorf("iterate trust provider ids: %w", err)
	}

	cpRows, err := r.pool.Query(ctx, `SELECT credential_provider_id, mapping_type, mapping_value FROM policy_credential_providers WHERE policy_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("load credential providers: %w", err)
	}
	defer cpRows.Close()
	for cpRows.Next() {
		var pcp domain.PolicyCredentialProvider
		if err := cpRows.Scan(&pcp.CredentialProviderID, &pcp.MappingType, &pcp.MappingValue); err != nil {
			return fmt.Errorf("scan credential provider mapping: %w", err)
		}
		p.CredentialProviderIDs = append(p.CredentialProviderIDs, pcp)
	}
	if err := cpRows.Err(); err != nil {
		return fmt.Errorf("iterate credential provider mappings: %w", err)
	}

	acRows, err := r.pool.Query(ctx, `SELECT access_condition_id FROM policy_access_conditions WHERE policy_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("load access conditions: %w", err)
	}
	defer acRows.Close()
	for acRows.Next() {
		var id uuid.UUID
		if err := acRows.Scan(&id); err != nil {
			return fmt.Errorf("scan access condition id: %w", err)
		}
		p.AccessConditionIDs = append(p.AccessConditionIDs, id)
	}
	if err := acRows.Err(); err != nil {
		return fmt.Errorf("iterate access condition ids: %w", err)
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

	tpRows, err := r.pool.Query(ctx, `SELECT policy_id, trust_provider_id FROM policy_trust_providers WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load trust providers: %w", err)
	}
	defer tpRows.Close()
	for tpRows.Next() {
		var policyID, tpID uuid.UUID
		if err := tpRows.Scan(&policyID, &tpID); err != nil {
			return fmt.Errorf("scan trust provider junction: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].TrustProviderIDs = append(policies[idx].TrustProviderIDs, tpID)
		}
	}
	if err := tpRows.Err(); err != nil {
		return fmt.Errorf("iterate trust provider junctions: %w", err)
	}

	cpRows, err := r.pool.Query(ctx, `SELECT policy_id, credential_provider_id, mapping_type, mapping_value FROM policy_credential_providers WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load credential providers: %w", err)
	}
	defer cpRows.Close()
	for cpRows.Next() {
		var policyID uuid.UUID
		var pcp domain.PolicyCredentialProvider
		if err := cpRows.Scan(&policyID, &pcp.CredentialProviderID, &pcp.MappingType, &pcp.MappingValue); err != nil {
			return fmt.Errorf("scan credential provider junction: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].CredentialProviderIDs = append(policies[idx].CredentialProviderIDs, pcp)
		}
	}
	if err := cpRows.Err(); err != nil {
		return fmt.Errorf("iterate credential provider junctions: %w", err)
	}

	acRows, err := r.pool.Query(ctx, `SELECT policy_id, access_condition_id FROM policy_access_conditions WHERE policy_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("batch load access conditions: %w", err)
	}
	defer acRows.Close()
	for acRows.Next() {
		var policyID, acID uuid.UUID
		if err := acRows.Scan(&policyID, &acID); err != nil {
			return fmt.Errorf("scan access condition junction: %w", err)
		}
		if idx, ok := idxMap[policyID]; ok {
			policies[idx].AccessConditionIDs = append(policies[idx].AccessConditionIDs, acID)
		}
	}
	if err := acRows.Err(); err != nil {
		return fmt.Errorf("iterate access condition junctions: %w", err)
	}

	return nil
}
