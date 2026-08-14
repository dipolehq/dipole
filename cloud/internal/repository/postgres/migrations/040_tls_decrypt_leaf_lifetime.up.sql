-- Migration 040: Add leaf certificate lifetime configuration for TLS Decrypt.
-- Stored on the tenants table alongside the existing CA cert/key columns (from migration 030).
-- Default 86400 seconds (1 day) matches Aembit's default.

ALTER TABLE tenants
    ADD COLUMN leaf_lifetime_seconds INT NOT NULL DEFAULT 86400;
