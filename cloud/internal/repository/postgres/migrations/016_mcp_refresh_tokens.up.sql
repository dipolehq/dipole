CREATE TABLE mcp_refresh_tokens (
    token TEXT PRIMARY KEY,
    client_id UUID NOT NULL REFERENCES mcp_oauth_clients(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_claims JSONB,
    policy_id UUID,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_mcp_refresh_tokens_client ON mcp_refresh_tokens(client_id);
CREATE INDEX idx_mcp_refresh_tokens_tenant ON mcp_refresh_tokens(tenant_id);
