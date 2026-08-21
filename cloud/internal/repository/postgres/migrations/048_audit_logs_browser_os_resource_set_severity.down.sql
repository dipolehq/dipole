DROP INDEX IF EXISTS idx_audit_logs_tenant_resource_set_time;
ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS resource_set_id,
    DROP COLUMN IF EXISTS os,
    DROP COLUMN IF EXISTS browser;
