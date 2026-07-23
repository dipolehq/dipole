ALTER TABLE tenants
    DROP COLUMN IF EXISTS ca_cert_pem,
    DROP COLUMN IF EXISTS ca_key_encrypted;
