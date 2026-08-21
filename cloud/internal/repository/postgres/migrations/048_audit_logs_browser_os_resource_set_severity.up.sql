-- Add browser, os, resource_set_id, and severity to audit_logs.
--
-- browser/os: the REST layer has always parsed these from the request's
-- User-Agent (see rest/audit_helpers.go's parseUA) but had no column to
-- store them in, so every audit log's Browser/OS was silently discarded
-- before reaching the database. The admin console's audit log detail
-- view already renders these fields; this makes that data real.
--
-- resource_set_id: access_authorization_events and workload_events are
-- both resource-set-scoped (see migration 011, 039); audit_logs was the
-- only event table left without it, meaning audit logs could never be
-- filtered/isolated per resource set the way every other event type is.
-- Defaulted to the Default resource set, matching the convention set by
-- migration 039 for the other two event tables.
--
-- severity: audit_logs has always had a severity column with a DEFAULT
-- of 'Info', but nothing in the Go code ever set it explicitly — every
-- row got 'Info' regardless of what happened. This migration doesn't
-- change the column (it already exists from migration 010); the
-- accompanying code change is what starts setting it meaningfully.

ALTER TABLE audit_logs
    ADD COLUMN browser TEXT,
    ADD COLUMN os TEXT,
    ADD COLUMN resource_set_id UUID NOT NULL DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

CREATE INDEX idx_audit_logs_tenant_resource_set_time
    ON audit_logs (tenant_id, resource_set_id, "timestamp" DESC);
