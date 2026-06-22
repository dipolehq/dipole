CREATE TABLE credential_provider_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL,
    config_encrypted BYTEA NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    tags JSONB NOT NULL DEFAULT '[]',
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cpi_tenant ON credential_provider_integrations(tenant_id);

ALTER TABLE credential_providers
    ADD CONSTRAINT fk_credential_providers_integration
    FOREIGN KEY (integration_id) REFERENCES credential_provider_integrations(id);
