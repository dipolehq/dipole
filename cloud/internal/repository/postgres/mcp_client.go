// mcp_client.go implements MCPClientRepository for PostgreSQL, storing dynamically registered MCP OAuth clients with their redirect URIs.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/service/mcp"
)

type MCPClientRepo struct {
	pool Pool
}

func NewMCPClientRepo(pool Pool) *MCPClientRepo {
	return &MCPClientRepo{pool: pool}
}

func (r *MCPClientRepo) Create(ctx context.Context, client *mcp.RegisteredClient) error {
	redirectURIs, _ := json.Marshal(client.RedirectURIs)
	grantTypes, _ := json.Marshal(client.GrantTypes)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO mcp_oauth_clients (id, tenant_id, redirect_uris, grant_types, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		client.ID, client.TenantID, redirectURIs, grantTypes, client.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create mcp client: %w", err)
	}
	return nil
}

func (r *MCPClientRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*mcp.RegisteredClient, error) {
	var client mcp.RegisteredClient
	var redirectURIsRaw, grantTypesRaw json.RawMessage

	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, redirect_uris, grant_types, created_at
		 FROM mcp_oauth_clients WHERE tenant_id = $1 AND id = $2`,
		tenantID, id,
	).Scan(&client.ID, &client.TenantID, &redirectURIsRaw, &grantTypesRaw, &client.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get mcp client: %w", err)
	}

	json.Unmarshal(redirectURIsRaw, &client.RedirectURIs)
	json.Unmarshal(grantTypesRaw, &client.GrantTypes)

	return &client, nil
}
