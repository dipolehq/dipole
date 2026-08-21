ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_resource_set_id_fkey;
ALTER TABLE role_resource_set_assignments DROP CONSTRAINT role_resource_set_assignments_resource_set_id_fkey;

ALTER TABLE resource_sets DROP CONSTRAINT resource_sets_pkey;
ALTER TABLE resource_sets ADD CONSTRAINT resource_sets_pkey PRIMARY KEY (id);

ALTER TABLE role_resource_set_assignments
    ADD CONSTRAINT role_resource_set_assignments_resource_set_id_fkey
    FOREIGN KEY (resource_set_id) REFERENCES resource_sets(id) ON DELETE CASCADE;
ALTER TABLE api_tokens
    ADD CONSTRAINT api_tokens_resource_set_id_fkey
    FOREIGN KEY (resource_set_id) REFERENCES resource_sets(id) ON DELETE SET NULL;

DROP INDEX IF EXISTS idx_role_resource_set_assignments_tenant;
ALTER TABLE role_resource_set_assignments DROP CONSTRAINT role_resource_set_assignments_tenant_id_fkey;
ALTER TABLE role_resource_set_assignments DROP COLUMN tenant_id;
