-- Migration 041: Add a session token hash to agent_proxies for per-call
-- caller authentication on the Agent Proxy <-> Cloud gRPC channel.
--
-- A session token is issued once at Register and its SHA-256 hash is
-- stored here. Every subsequent RPC (Evaluate, SignLeafCertificate,
-- ReportHealth, IngestEvents) must present this token via gRPC metadata;
-- Cloud looks it up by hash to resolve the caller's verified identity
-- instead of trusting the tenant_id/proxy_id fields in the request body.
-- Re-registering overwrites the hash, invalidating the previous token.

ALTER TABLE agent_proxies
    ADD COLUMN session_token_hash TEXT;

CREATE UNIQUE INDEX idx_agent_proxies_session_token_hash
    ON agent_proxies (session_token_hash)
    WHERE session_token_hash IS NOT NULL;
