CREATE TABLE mcp_authorization_codes (
    code TEXT PRIMARY KEY,
    client_id UUID NOT NULL REFERENCES mcp_oauth_clients(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    redirect_uri TEXT NOT NULL,
    code_challenge TEXT NOT NULL,
    code_challenge_method TEXT NOT NULL DEFAULT 'S256',
    scope TEXT NOT NULL DEFAULT 'openid',
    user_claims JSONB,
    policy_id UUID,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX idx_mcp_auth_codes_client ON mcp_authorization_codes(client_id);
CREATE INDEX idx_mcp_auth_codes_tenant ON mcp_authorization_codes(tenant_id);
