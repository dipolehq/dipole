CREATE TABLE mcp_user_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    credential_provider_id UUID NOT NULL,
    user_id TEXT NOT NULL,
    mcp_server_workload_id UUID NOT NULL,
    access_token_encrypted BYTEA NOT NULL,
    refresh_token_encrypted BYTEA,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, credential_provider_id, user_id, mcp_server_workload_id)
);

CREATE INDEX idx_mcp_user_creds_lookup ON mcp_user_credentials(tenant_id, user_id, mcp_server_workload_id);
