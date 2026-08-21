DROP INDEX IF EXISTS idx_agent_proxies_session_token_hash;

ALTER TABLE agent_proxies
    DROP COLUMN IF EXISTS session_token_hash;
