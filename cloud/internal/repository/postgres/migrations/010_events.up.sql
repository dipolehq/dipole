CREATE TABLE access_authorization_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    context_id UUID NOT NULL,
    severity TEXT NOT NULL DEFAULT 'Info',
    client_ip TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL,
    resource_set_id UUID
) PARTITION BY RANGE (timestamp);

CREATE TABLE access_authorization_events_default PARTITION OF access_authorization_events DEFAULT;

CREATE INDEX idx_auth_events_tenant_time ON access_authorization_events(tenant_id, timestamp DESC);
CREATE INDEX idx_auth_events_context ON access_authorization_events(context_id);

CREATE TABLE workload_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    connection_id UUID NOT NULL,
    event_type TEXT NOT NULL,
    policy_id UUID,
    client_workload_id UUID,
    server_workload_id UUID,
    outcome TEXT,
    severity TEXT NOT NULL DEFAULT 'Info',
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    payload JSONB NOT NULL,
    resource_set_id UUID
) PARTITION BY RANGE (timestamp);

CREATE TABLE workload_events_default PARTITION OF workload_events DEFAULT;

CREATE INDEX idx_workload_events_tenant_time ON workload_events(tenant_id, timestamp DESC);
CREATE INDEX idx_workload_events_connection ON workload_events(connection_id);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    actor_id UUID,
    actor_email TEXT,
    category TEXT NOT NULL,
    activity TEXT NOT NULL,
    target_type TEXT,
    target_id UUID,
    target_name TEXT,
    severity TEXT NOT NULL DEFAULT 'Info',
    client_ip TEXT,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_tenant_time ON audit_logs(tenant_id, timestamp DESC);
