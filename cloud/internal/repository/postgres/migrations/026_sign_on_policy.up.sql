CREATE TABLE IF NOT EXISTS sign_on_policies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    require_sso boolean NOT NULL DEFAULT false,
    require_mfa boolean NOT NULL DEFAULT false,
    mfa_grace_period_hours integer NOT NULL DEFAULT 24,
    auth_timeout_minutes integer NOT NULL DEFAULT 480,
    created_at timestamp with time zone NOT NULL DEFAULT now(),
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_sign_on_policies_tenant_unique ON sign_on_policies(tenant_id);
