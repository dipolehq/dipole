ALTER TABLE tenants
    ADD COLUMN ca_cert_pem TEXT,
    ADD COLUMN ca_key_encrypted BYTEA;
