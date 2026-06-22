package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TrustProviderRepo struct {
	pool Pool
}

func (r *TrustProviderRepo) Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error {
	tp.ID = uuid.New()
	tp.TenantID = tenantID

	matchRules, _ := json.Marshal(tp.MatchRules)
	tags, _ := json.Marshal(tp.Tags)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO trust_providers (id, tenant_id, name, description, is_active, type, config, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		tp.ID, tenantID, tp.Name, tp.Description, tp.IsActive, tp.Type, tp.Config, matchRules, tp.EdgeSDKClientID, tp.EdgeSDKAudience, tags, tp.ResourceSetID)
	if err != nil {
		return fmt.Errorf("create trust provider: %w", err)
	}
	return nil
}

func (r *TrustProviderRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.TrustProvider, error) {
	tp := &domain.TrustProvider{}
	var matchRules, tags []byte

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id, created_at, updated_at
		 FROM trust_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id).
		Scan(&tp.ID, &tp.TenantID, &tp.Name, &tp.Description, &tp.IsActive, &tp.Type, &tp.Config, &matchRules, &tp.EdgeSDKClientID, &tp.EdgeSDKAudience, &tags, &tp.ResourceSetID, &tp.CreatedAt, &tp.UpdatedAt)
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
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM trust_providers WHERE tenant_id = $1`, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count trust providers: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, description, is_active, type, config, match_rules, edge_sdk_client_id, edge_sdk_audience, tags, resource_set_id, created_at, updated_at
		 FROM trust_providers WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, opts.PerPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list trust providers: %w", err)
	}
	defer rows.Close()

	var results []domain.TrustProvider
	for rows.Next() {
		var tp domain.TrustProvider
		var matchRules, tags []byte
		if err := rows.Scan(&tp.ID, &tp.TenantID, &tp.Name, &tp.Description, &tp.IsActive, &tp.Type, &tp.Config, &matchRules, &tp.EdgeSDKClientID, &tp.EdgeSDKAudience, &tags, &tp.ResourceSetID, &tp.CreatedAt, &tp.UpdatedAt); err != nil {
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

	_, err := r.pool.Exec(ctx,
		`UPDATE trust_providers SET name=$1, description=$2, is_active=$3, type=$4, config=$5, match_rules=$6, edge_sdk_client_id=$7, edge_sdk_audience=$8, tags=$9, resource_set_id=$10, updated_at=NOW()
		 WHERE tenant_id = $11 AND id = $12`,
		tp.Name, tp.Description, tp.IsActive, tp.Type, tp.Config, matchRules, tp.EdgeSDKClientID, tp.EdgeSDKAudience, tags, tp.ResourceSetID, tenantID, tp.ID)
	if err != nil {
		return fmt.Errorf("update trust provider: %w", err)
	}
	return nil
}

func (r *TrustProviderRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM trust_providers WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return fmt.Errorf("delete trust provider: %w", err)
	}
	return nil
}
