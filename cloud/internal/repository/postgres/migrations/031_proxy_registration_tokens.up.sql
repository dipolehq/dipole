CREATE TABLE proxy_registration_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    token_hash TEXT NOT NULL UNIQUE,
    controller_id UUID NOT NULL,
    consumed BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_proxy_reg_tokens_hash ON proxy_registration_tokens(token_hash);
CREATE INDEX idx_proxy_reg_tokens_tenant ON proxy_registration_tokens(tenant_id);
