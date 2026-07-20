CREATE TABLE IF NOT EXISTS global_policy_compliance (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    ap_trust_provider text NOT NULL DEFAULT 'required',
    ap_access_condition text NOT NULL DEFAULT 'recommended',
    ac_trust_provider text NOT NULL DEFAULT 'recommended',
    ac_tls_hostname text NOT NULL DEFAULT 'recommended',
    access_token_lifetime_seconds integer NOT NULL DEFAULT 604800,
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);
