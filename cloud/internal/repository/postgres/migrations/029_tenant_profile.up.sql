CREATE TABLE IF NOT EXISTS tenant_profiles (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    first_name text NOT NULL DEFAULT '',
    last_name text NOT NULL DEFAULT '',
    email text NOT NULL DEFAULT '',
    phone text NOT NULL DEFAULT '',
    updated_at timestamp with time zone NOT NULL DEFAULT now()
);
