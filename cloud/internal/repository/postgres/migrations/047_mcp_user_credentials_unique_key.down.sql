ALTER TABLE mcp_user_credentials
    DROP CONSTRAINT mcp_user_credentials_tenant_user_server_key;

-- Restores the original 4-column uniqueness guarantee. Uses an explicit
-- name (rather than an unnamed inline UNIQUE(...), which is how migration
-- 017 originally declared it) so the name is deterministic instead of
-- Postgres auto-generating and silently truncating one; the columns and
-- behavior are identical to the original constraint either way.
ALTER TABLE mcp_user_credentials
    ADD CONSTRAINT mcp_user_credentials_orig_unique_key
    UNIQUE (tenant_id, credential_provider_id, user_id, mcp_server_workload_id);
