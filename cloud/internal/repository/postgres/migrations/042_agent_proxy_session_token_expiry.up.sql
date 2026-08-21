-- Migration 042: Add an expiry to the Agent Proxy session token.
--
-- Migration 041 introduced session_token_hash for per-call caller
-- authentication but the token never expired. This adds an expiry so a
-- session token issued at Register (or RefreshSessionToken) is only valid
-- for a bounded window, closing a long-lived-credential gap: the proxy
-- must periodically call RefreshSessionToken before this expires, or its
-- session token stops authenticating any further RPCs.

ALTER TABLE agent_proxies
    ADD COLUMN session_token_expires_at TIMESTAMPTZ;
