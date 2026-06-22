CREATE TABLE trust_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    type TEXT NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    match_rules JSONB NOT NULL DEFAULT '[]',
    edge_sdk_client_id TEXT UNIQUE,
    edge_sdk_audience TEXT,
    tags JSONB NOT NULL DEFAULT '[]',
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trust_providers_tenant ON trust_providers(tenant_id);
