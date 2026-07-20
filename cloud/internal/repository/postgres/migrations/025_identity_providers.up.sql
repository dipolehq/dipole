CREATE TABLE IF NOT EXISTS identity_providers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    is_active boolean NOT NULL DEFAULT true,
    type text NOT NULL DEFAULT '',
    config jsonb NOT NULL DEFAULT '{}',
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_identity_providers_tenant ON identity_providers(tenant_id);

-- Add columns that may be missing from older schema versions
ALTER TABLE identity_providers ADD COLUMN IF NOT EXISTS description text NOT NULL DEFAULT '';
ALTER TABLE identity_providers ADD COLUMN IF NOT EXISTS is_active boolean NOT NULL DEFAULT true;
