-- Migration 043: Add a shared_secret column to agent_controllers.
--
-- RegisterController has generated a Kerberos-attestation HMAC signing
-- secret since it was written, but no column ever existed to persist it:
-- AgentControllerRepo.Update's UPDATE statement never listed it, so the
-- secret handed to the Controller at registration was silently discarded
-- and could never match anything read back later. Kerberos trust-provider
-- attestation has been non-functional as a result. Stored encrypted at
-- rest (AES-256-GCM via internal/crypto), matching every other secret in
-- this codebase (MFA TOTP secrets, credential provider configs, tenant CA
-- private keys) — never stored in plaintext.

ALTER TABLE agent_controllers
    ADD COLUMN shared_secret_encrypted BYTEA;
