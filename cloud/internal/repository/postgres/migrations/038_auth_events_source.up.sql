-- Add source discriminator to access_authorization_events.
-- Identifies whether the event originated from the Edge Proxy's policy
-- evaluation pipeline or the MCP Identity Gateway's authorization flow.
-- Default 'edge_proxy' makes this migration additive: all existing rows
-- get the correct value without a backfill, and the existing Edge Proxy
-- emission path doesn't need a code change to produce correct data.

ALTER TABLE access_authorization_events
    ADD COLUMN source TEXT NOT NULL DEFAULT 'edge_proxy';

-- Composite index for the MCP Logs page query pattern:
-- WHERE tenant_id = $1 AND source = 'mcp_gateway' ORDER BY timestamp DESC
CREATE INDEX idx_auth_events_tenant_source_time
    ON access_authorization_events (tenant_id, source, "timestamp" DESC);
