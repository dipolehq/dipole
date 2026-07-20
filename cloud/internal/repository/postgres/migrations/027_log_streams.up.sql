CREATE TABLE IF NOT EXISTS log_streams (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT true,
    event_type text NOT NULL DEFAULT '',
    destination_type text NOT NULL DEFAULT '',
    config jsonb NOT NULL DEFAULT '{}',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_log_streams_tenant ON log_streams(tenant_id);
