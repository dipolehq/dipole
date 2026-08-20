-- Expand step of an expand-migrate-contract migration: adds encrypted
-- storage for log_streams.config (currently plaintext JSONB, holds
-- Splunk HEC tokens, CrowdStrike client secrets, and will hold AWS/GCS
-- credentials) plus export failure visibility. The plaintext `config`
-- column is intentionally kept for now; a follow-up migration drops it
-- once the application-level backfill into config_encrypted has been
-- confirmed in every environment.
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS config_encrypted bytea;
ALTER TABLE log_streams ADD COLUMN IF NOT EXISTS last_export_error text;
