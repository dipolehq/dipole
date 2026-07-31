/*
api_token.go implements the APITokenRepository for PostgreSQL.
Supports multiple tokens per tenant, each optionally scoped to a Resource Set.
*/
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type APITokenRepo struct {
	pool Pool
}

func (r *APITokenRepo) Create(ctx context.Context, tenantID uuid.UUID, token *domain.APIToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	token.TenantID = tenantID

	err := r.pool.QueryRow(ctx,
		`INSERT INTO api_tokens (id, tenant_id, name, token_hash, resource_set_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		token.ID, tenantID, token.Name, token.TokenHash, token.ResourceSetID).
		Scan(&token.CreatedAt)
	if err != nil {
		return fmt.Errorf("create api token: %w", err)
	}
	return nil
}

func (r *APITokenRepo) GetByHash(ctx context.Context, tokenHash string) (*domain.APIToken, error) {
	t := &domain.APIToken{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, token_hash, resource_set_id, created_at
		 FROM api_tokens WHERE token_hash = $1`,
		tokenHash).
		Scan(&t.ID, &t.TenantID, &t.Name, &t.TokenHash, &t.ResourceSetID, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get api token by hash: %w", err)
	}
	return t, nil
}

func (r *APITokenRepo) List(ctx context.Context, tenantID uuid.UUID) ([]domain.APIToken, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, token_hash, resource_set_id, created_at
		 FROM api_tokens WHERE tenant_id = $1
		 ORDER BY created_at DESC`,
		tenantID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	defer rows.Close()

	var results []domain.APIToken
	for rows.Next() {
		var t domain.APIToken
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.TokenHash, &t.ResourceSetID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan api token: %w", err)
		}
		results = append(results, t)
	}
	return results, nil
}

func (r *APITokenRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM api_tokens WHERE tenant_id = $1 AND id = $2`,
		tenantID, id)
	if err != nil {
		return fmt.Errorf("delete api token: %w", err)
	}
	return nil
}
