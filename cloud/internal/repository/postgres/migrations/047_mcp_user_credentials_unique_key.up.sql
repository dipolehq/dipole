-- mcp_user_credentials' original unique constraint was
-- (tenant_id, credential_provider_id, user_id, mcp_server_workload_id),
-- but MCPUserCredentialRepo.Get has always looked up a token by
-- (tenant_id, user_id, mcp_server_workload_id) alone — credential_provider_id
-- is never part of the read path. That mismatch meant Store's ON CONFLICT
-- (keyed on the full 4-column constraint) could never correctly upsert
-- against what Get treats as "the same token": if credential_provider_id
-- ever differed between two Store calls for the same tenant/user/server
-- (e.g. a policy's credential provider is swapped and the user re-consents),
-- a second row would be inserted instead of the existing one being updated,
-- and Get would then have two matching rows to choose from non-deterministically.
--
-- This migration narrows the unique constraint to the 3 columns the
-- application actually treats as the identity of a per-user MCP
-- credential, matching idx_mcp_user_creds_lookup (already indexed on
-- exactly these 3 columns) and MCPUserCredentialRepo.Get/Store's real
-- behavior. credential_provider_id remains a plain (non-unique) column
-- for record-keeping.
--
-- The original constraint was declared inline (table-level UNIQUE(...))
-- in migration 017 without an explicit name, so Postgres auto-generated
-- its name; rather than hardcode a guess at that generated name (which
-- can be truncated/deduplicated by Postgres in ways that aren't safe to
-- predict by hand), this looks it up by its actual columns and drops it
-- by its real name.

DO $$
DECLARE
    old_constraint_name TEXT;
BEGIN
    -- Compare column *sets* (not conkey's declaration order) so this
    -- matches regardless of how migration 017 ordered the columns in
    -- its inline UNIQUE(...) clause.
    SELECT con.conname INTO old_constraint_name
    FROM pg_constraint con
    JOIN pg_class rel ON rel.oid = con.conrelid
    WHERE rel.relname = 'mcp_user_credentials'
      AND con.contype = 'u'
      AND (
          SELECT array_agg(attname::text ORDER BY attname)
          FROM pg_attribute
          WHERE attrelid = rel.oid AND attnum = ANY (con.conkey)
      ) = ARRAY['credential_provider_id', 'mcp_server_workload_id', 'tenant_id', 'user_id'];

    IF old_constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE mcp_user_credentials DROP CONSTRAINT %I', old_constraint_name);
    END IF;
END $$;

ALTER TABLE mcp_user_credentials
    ADD CONSTRAINT mcp_user_credentials_tenant_user_server_key
    UNIQUE (tenant_id, user_id, mcp_server_workload_id);
