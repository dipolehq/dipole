CREATE TABLE server_workloads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    host TEXT NOT NULL,
    port INT NOT NULL,
    app_protocol TEXT NOT NULL DEFAULT 'HTTP',
    tls BOOLEAN NOT NULL DEFAULT true,
    auth_method TEXT NOT NULL DEFAULT 'http_bearer',
    auth_scheme TEXT NOT NULL DEFAULT 'Bearer',
    auth_config JSONB NOT NULL DEFAULT '{}',
    url_path TEXT NOT NULL DEFAULT '',
    tags JSONB NOT NULL DEFAULT '[]',
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_server_workloads_tenant ON server_workloads(tenant_id);
CREATE INDEX idx_server_workloads_target ON server_workloads(tenant_id, host, port);
