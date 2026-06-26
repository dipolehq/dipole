/*
mcp_user_credential.go implements per-user OAuth token storage for MCP servers.
Tokens are encrypted at rest (AES-256-GCM). Lookups are always scoped by
tenant_id + user_id + server_workload_id to enforce per-user isolation.
*/
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
)

type MCPUserCredentialRepo struct {
	pool   *pgxpool.Pool
	encKey []byte
}

func NewMCPUserCredentialRepo(pool *pgxpool.Pool, encKey []byte) *MCPUserCredentialRepo {
	return &MCPUserCredentialRepo{pool: pool, encKey: encKey}
}

func (r *MCPUserCredentialRepo) Get(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID) (*gw.UserTokenEntry, error) {
	var accessEnc, refreshEnc []byte
	var expiresAt time.Time

	err := r.pool.QueryRow(ctx,
		`SELECT access_token_encrypted, refresh_token_encrypted, expires_at
		 FROM mcp_user_credentials
		 WHERE tenant_id = $1 AND user_id = $2 AND mcp_server_workload_id = $3`,
		tenantID, userID, serverWorkloadID,
	).Scan(&accessEnc, &refreshEnc, &expiresAt)
	if err != nil {
		return nil, nil
	}

	accessToken, err := Decrypt(accessEnc, r.encKey)
	if err != nil {
		return nil, err
	}

	var refreshToken string
	if len(refreshEnc) > 0 {
		rt, err := Decrypt(refreshEnc, r.encKey)
		if err == nil {
			refreshToken = string(rt)
		}
	}

	return &gw.UserTokenEntry{
		AccessToken:  string(accessToken),
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func (r *MCPUserCredentialRepo) Store(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, token *gw.UserTokenEntry) error {
	accessEnc, err := Encrypt([]byte(token.AccessToken), r.encKey)
	if err != nil {
		return err
	}

	var refreshEnc []byte
	if token.RefreshToken != "" {
		refreshEnc, err = Encrypt([]byte(token.RefreshToken), r.encKey)
		if err != nil {
			return err
		}
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO mcp_user_credentials (tenant_id, credential_provider_id, user_id, mcp_server_workload_id, access_token_encrypted, refresh_token_encrypted, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, credential_provider_id, user_id, mcp_server_workload_id)
		 DO UPDATE SET access_token_encrypted = $5, refresh_token_encrypted = $6, expires_at = $7, updated_at = NOW()`,
		tenantID, uuid.Nil, userID, serverWorkloadID, accessEnc, refreshEnc, token.ExpiresAt,
	)
	return err
}
