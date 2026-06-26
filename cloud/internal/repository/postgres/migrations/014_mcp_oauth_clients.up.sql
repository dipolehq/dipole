CREATE TABLE mcp_oauth_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    redirect_uris JSONB NOT NULL,
    grant_types JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mcp_oauth_clients_tenant ON mcp_oauth_clients(tenant_id);
