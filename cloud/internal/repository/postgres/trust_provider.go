// trust_provider.go implements the TrustProviderRepository for PostgreSQL.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TrustProviderRepo struct {
	pool   Pool
	encKey []byte
}

func (r *TrustProviderRepo) Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	if tp.ID == uuid.Nil {
		tp.ID = uuid.New()
	}
	tp.TenantID = tenantID

	matchRules, _ := json.Marshal(tp.MatchRules)
	tags, _ := json.Marshal(tp.Tags)

	err := r.pool.QueryRow(ctx,
		`INSERT INTO trust_providers (id, tenant_id, name, description, is_active, type, config, symmetric_key_encrypted, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		 RETURNING created_at, updated_at`,
		tp.ID, tenantID, tp.Name, tp.Description, tp.IsActive, tp.Type, tp.Config, tp.SymmetricKeyEncrypted, matchRules, tp.EdgeSDKClientID, tp.EdgeSDKAudience, tags, tp.ResourceSetID).
		Scan(&tp.CreatedAt, &tp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create trust provider: %w", err)
	}
	return nil
}

func (r *TrustProviderRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp := &domain.TrustProvider{}
	var matchRules, tags []byte

	args := []interface{}{tenantID, id}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, symmetric_key_encrypted, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id, created_at, updated_at
		 FROM trust_providers WHERE tenant_id = $1 AND id = $2`+rsFilter, args...).
		Scan(&tp.ID, &tp.TenantID, &tp.Name, &tp.Description, &tp.IsActive, &tp.Type, &tp.Config, &tp.SymmetricKeyEncrypted, &matchRules, &tp.EdgeSDKClientID, &tp.EdgeSDKAudience, &tags, &tp.ResourceSetID, &tp.CreatedAt, &tp.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get trust provider: %w", err)
	}

	json.Unmarshal(matchRules, &tp.MatchRules)
	json.Unmarshal(tags, &tp.Tags)
	return tp, nil
}

func (r *TrustProviderRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.TrustProvider, int, error) {
	where, countArgs, n := buildTenantSearchFilter(tenantID, opts.Search)
	rsFilter := ResourceSetFilter(ctx, &countArgs, &n)

	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM trust_providers WHERE `+where+rsFilter,
		countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count trust providers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	where, listArgs, n := buildTenantSearchFilter(tenantID, opts.Search)
	rsFilter = ResourceSetFilter(ctx, &listArgs, &n)
	limitOffset := appendLimitOffset(&listArgs, &n, opts.PerPage, offset)

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, symmetric_key_encrypted, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id, created_at, updated_at,
		        (SELECT COUNT(*) FROM policy_trust_providers ptp WHERE ptp.trust_provider_id = trust_providers.id) AS policy_count
		 FROM trust_providers
		 WHERE `+where+rsFilter+`
		 ORDER BY created_at DESC `+limitOffset,
		listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list trust providers: %w", err)
	}
	defer rows.Close()

	var results []domain.TrustProvider
	for rows.Next() {
		var tp domain.TrustProvider
		var matchRules, tags []byte
		if err := rows.Scan(&tp.ID, &tp.TenantID, &tp.Name, &tp.Description, &tp.IsActive, &tp.Type, &tp.Config, &tp.SymmetricKeyEncrypted, &matchRules, &tp.EdgeSDKClientID, &tp.EdgeSDKAudience, &tags, &tp.ResourceSetID, &tp.CreatedAt, &tp.UpdatedAt, &tp.PolicyCount); err != nil {
			return nil, 0, fmt.Errorf("scan trust provider: %w", err)
		}
		json.Unmarshal(matchRules, &tp.MatchRules)
		json.Unmarshal(tags, &tp.Tags)
		results = append(results, tp)
	}
	return results, total, nil
}

func (r *TrustProviderRepo) Update(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	matchRules, _ := json.Marshal(tp.MatchRules)
	tags, _ := json.Marshal(tp.Tags)

	args := []interface{}{tp.Name, tp.Description, tp.IsActive, tp.Type, tp.Config, tp.SymmetricKeyEncrypted, matchRules, tp.EdgeSDKClientID, tp.EdgeSDKAudience, tags, tp.ResourceSetID, tenantID, tp.ID}
	n := 13
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	err := r.pool.QueryRow(ctx,
		`UPDATE trust_providers SET name=$1, description=$2, is_active=$3, type=$4, config=$5, symmetric_key_encrypted=$6, match_rules=$7, edge_sdk_client_id=$8, edge_sdk_audience=$9, tags=$10, resource_set_id=$11, updated_at=NOW()
		 WHERE tenant_id = $12 AND id = $13`+rsFilter+`
		 RETURNING created_at, updated_at`,
		args...).
		Scan(&tp.CreatedAt, &tp.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update trust provider: %w", err)
	}
	return nil
}

func (r *TrustProviderRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	args := []interface{}{tenantID, id}
	n := 2
	rsFilter := ResourceSetFilter(ctx, &args, &n)

	_, err := r.pool.Exec(ctx, `DELETE FROM trust_providers WHERE tenant_id = $1 AND id = $2`+rsFilter, args...)
	if err != nil {
		return fmt.Errorf("delete trust provider: %w", err)
	}
	return nil
}

func (r *TrustProviderRepo) CountInactive(ctx context.Context, tenantID uuid.UUID) int {
	var count int
	_ = r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM trust_providers WHERE tenant_id = $1 AND is_active = false`,
		tenantID).Scan(&count)
	return count
}

func (r *TrustProviderRepo) BackfillSymmetricKeyEncryption(ctx context.Context) (int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, config FROM trust_providers WHERE symmetric_key_encrypted IS NULL AND config::jsonb ? 'symmetricKey'`)
	if err != nil {
		return 0, fmt.Errorf("query unmigrated trust providers: %w", err)
	}

	type pending struct {
		id     uuid.UUID
		config []byte
	}
	var toMigrate []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.id, &p.config); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan unmigrated trust provider: %w", err)
		}
		toMigrate = append(toMigrate, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate unmigrated trust providers: %w", err)
	}

	migrated := 0
	for _, p := range toMigrate {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(p.config, &raw); err != nil {
			return migrated, fmt.Errorf("parse config for trust provider %s: %w", p.id, err)
		}

		rawKey, ok := raw["symmetricKey"]
		if !ok {
			continue
		}
		var symmetricKey string
		if err := json.Unmarshal(rawKey, &symmetricKey); err != nil {
			return migrated, fmt.Errorf("parse symmetricKey for trust provider %s: %w", p.id, err)
		}

		delete(raw, "symmetricKey")
		strippedConfig, err := json.Marshal(raw)
		if err != nil {
			return migrated, fmt.Errorf("re-marshal config for trust provider %s: %w", p.id, err)
		}

		var encrypted []byte
		if symmetricKey != "" {
			encrypted, err = crypto.Encrypt([]byte(symmetricKey), r.encKey)
			if err != nil {
				return migrated, fmt.Errorf("encrypt symmetricKey for trust provider %s: %w", p.id, err)
			}
		}

		tag, err := r.pool.Exec(ctx,
			`UPDATE trust_providers SET config = $1, symmetric_key_encrypted = $2
			 WHERE id = $3 AND symmetric_key_encrypted IS NULL AND config IS NOT DISTINCT FROM $4`,
			strippedConfig, encrypted, p.id, p.config)
		if err != nil {
			return migrated, fmt.Errorf("backfill symmetric_key_encrypted for trust provider %s: %w", p.id, err)
		}
		if tag.RowsAffected() == 1 {
			migrated++
		}
	}

	return migrated, nil
}
