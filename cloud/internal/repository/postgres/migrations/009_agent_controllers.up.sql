CREATE TABLE agent_controllers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'Unregistered',
    device_code TEXT,
    device_code_expires_at TIMESTAMPTZ,
    trust_provider_id UUID REFERENCES trust_providers(id),
    last_health_at TIMESTAMPTZ,
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_controllers_tenant ON agent_controllers(tenant_id);

CREATE TABLE agent_proxies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    agent_controller_id UUID REFERENCES agent_controllers(id),
    status TEXT NOT NULL DEFAULT 'Unregistered',
    last_health_at TIMESTAMPTZ,
    resource_set_id UUID,
    registered_at TIMESTAMPTZ
);

CREATE INDEX idx_agent_proxies_tenant ON agent_proxies(tenant_id);
