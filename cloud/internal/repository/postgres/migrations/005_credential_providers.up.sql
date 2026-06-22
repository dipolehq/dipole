CREATE TABLE credential_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    type TEXT NOT NULL,
    config_encrypted BYTEA NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]',
    integration_id UUID,
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_credential_providers_tenant ON credential_providers(tenant_id);
