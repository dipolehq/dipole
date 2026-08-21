// mcp_user_credential.go implements per-user OAuth token storage for MCP servers, encrypted at rest (AES-256-GCM) and scoped by tenant_id + user_id + server_workload_id.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
)

type MCPUserCredentialRepo struct {
	pool   Pool
	encKey []byte
}

func NewMCPUserCredentialRepo(pool Pool, encKey []byte) *MCPUserCredentialRepo {
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
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get mcp user credential: %w", err)
	}

	accessToken, err := crypto.Decrypt(accessEnc, r.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt mcp user access token: %w", err)
	}

	var refreshToken string
	if len(refreshEnc) > 0 {
		rt, err := crypto.Decrypt(refreshEnc, r.encKey)
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

func (r *MCPUserCredentialRepo) Store(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, credentialProviderID uuid.UUID, token *gw.UserTokenEntry) error {
	accessEnc, err := crypto.Encrypt([]byte(token.AccessToken), r.encKey)
	if err != nil {
		return fmt.Errorf("encrypt mcp user access token: %w", err)
	}

	var refreshEnc []byte
	if token.RefreshToken != "" {
		refreshEnc, err = crypto.Encrypt([]byte(token.RefreshToken), r.encKey)
		if err != nil {
			return fmt.Errorf("encrypt mcp user refresh token: %w", err)
		}
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO mcp_user_credentials (tenant_id, credential_provider_id, user_id, mcp_server_workload_id, access_token_encrypted, refresh_token_encrypted, expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (tenant_id, user_id, mcp_server_workload_id)
		 DO UPDATE SET credential_provider_id = $2, access_token_encrypted = $5, refresh_token_encrypted = $6, expires_at = $7, updated_at = NOW()`,
		tenantID, credentialProviderID, userID, serverWorkloadID, accessEnc, refreshEnc, token.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("store mcp user credential: %w", err)
	}
	return nil
}
