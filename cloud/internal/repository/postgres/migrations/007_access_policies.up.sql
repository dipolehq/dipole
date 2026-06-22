CREATE TABLE access_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT true,
    client_workload_id UUID NOT NULL REFERENCES client_workloads(id),
    server_workload_id UUID NOT NULL REFERENCES server_workloads(id),
    tags JSONB NOT NULL DEFAULT '[]',
    resource_set_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_policies_tenant_active ON access_policies(tenant_id, is_active);
CREATE INDEX idx_access_policies_server ON access_policies(tenant_id, server_workload_id);

CREATE TABLE policy_trust_providers (
    policy_id UUID NOT NULL REFERENCES access_policies(id) ON DELETE CASCADE,
    trust_provider_id UUID NOT NULL REFERENCES trust_providers(id) ON DELETE CASCADE,
    PRIMARY KEY (policy_id, trust_provider_id)
);

CREATE TABLE policy_credential_providers (
    policy_id UUID NOT NULL REFERENCES access_policies(id) ON DELETE CASCADE,
    credential_provider_id UUID NOT NULL REFERENCES credential_providers(id) ON DELETE CASCADE,
    mapping_type TEXT NOT NULL DEFAULT 'None',
    mapping_value TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (policy_id, credential_provider_id)
);

CREATE TABLE policy_access_conditions (
    policy_id UUID NOT NULL REFERENCES access_policies(id) ON DELETE CASCADE,
    access_condition_id UUID NOT NULL REFERENCES access_conditions(id) ON DELETE CASCADE,
    PRIMARY KEY (policy_id, access_condition_id)
);
