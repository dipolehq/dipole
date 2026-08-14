DROP INDEX IF EXISTS idx_auth_events_tenant_source_time;
ALTER TABLE access_authorization_events DROP COLUMN IF EXISTS source;
