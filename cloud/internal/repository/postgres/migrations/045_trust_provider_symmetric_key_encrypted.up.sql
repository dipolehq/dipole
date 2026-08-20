-- Expand step of an expand-migrate-contract migration: adds encrypted
-- storage for the OIDC symmetric_key attestation secret, which today is
-- stored in plaintext inside trust_providers.config (JSONB). The plaintext
-- `config` column is intentionally kept for its non-secret fields (OIDC
-- endpoint, JWKS, match rules, etc); only the symmetricKey value is
-- migrated out of it and into this encrypted sibling column, backfilled by
-- application code (encryption requires the Go-side key, not doable in
-- raw SQL) and then stripped from the plaintext config.
ALTER TABLE trust_providers ADD COLUMN IF NOT EXISTS symmetric_key_encrypted bytea;
