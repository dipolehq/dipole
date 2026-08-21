-- Fixes two bugs found while verifying RBAC end to end:
--
-- 1. resource_sets' primary key was (id) alone, not scoped by tenant.
--    Every tenant's "Default" resource set is created with the same
--    well-known UUID (ffffffff-ffff-ffff-ffff-ffffffffffff, see
--    scope.DefaultResourceSetID) — every resource-scoped table in the
--    schema already has a column DEFAULT pointing at this exact literal.
--    With a single-column PK, only the very first tenant in the database
--    can ever hold that UUID; every subsequent tenant's signup fails with
--    a duplicate-key error on resource_sets_pkey. Changing the primary
--    key to (tenant_id, id) lets every tenant own an independent row with
--    that same well-known ID, which is what the rest of the schema/code
--    already assumes.
--
-- 2. role_resource_set_assignments had no tenant_id column at all, and
--    its lookups (RoleRepo.GetResourceSetIDs, ResourceSetRepo.GetRoleIDs)
--    filtered only by role_id/resource_set_id — a role/resource-set pair
--    could theoretically be looked up across tenant boundaries if IDs
--    ever collided. Unused by any call site today (confirmed via
--    repo-wide search), but fixed now while this table is already being
--    touched rather than left as a latent gap for whoever wires it up.
--
-- Both foreign keys that reference resource_sets(id) (api_tokens,
-- role_resource_set_assignments) are updated to reference the new
-- composite (tenant_id, id) so a row can never reference another
-- tenant's resource set.

-- --- Step 1: drop the two FKs that depend on resource_sets_pkey ---
-- (Postgres refuses to drop a constraint that a foreign key still
-- depends on; both are recreated against the new composite key in
-- Step 3 below.)

ALTER TABLE role_resource_set_assignments DROP CONSTRAINT role_resource_set_assignments_resource_set_id_fkey;
ALTER TABLE api_tokens DROP CONSTRAINT api_tokens_resource_set_id_fkey;

-- --- Step 2a: resource_sets primary key -> (tenant_id, id) ---

ALTER TABLE resource_sets DROP CONSTRAINT resource_sets_pkey;
ALTER TABLE resource_sets ADD CONSTRAINT resource_sets_pkey PRIMARY KEY (tenant_id, id);
-- id remains globally unique in every table that currently exists (each
-- row's id is a real gen_random_uuid() with the sole exception of the
-- well-known Default id, which is now the one case the composite key
-- exists to allow to repeat per tenant), but a UNIQUE constraint on id
-- alone would defeat the entire point of this migration, so none is added.

-- --- Step 2b: role_resource_set_assignments gains tenant scoping ---

ALTER TABLE role_resource_set_assignments ADD COLUMN tenant_id UUID;

UPDATE role_resource_set_assignments rrsa
SET tenant_id = r.tenant_id
FROM roles r
WHERE r.id = rrsa.role_id;

-- No production data has ever populated this table via the application
-- (AssignResourceSets is currently unused by any handler), so the backfill
-- above is a no-op in every real environment today; it exists so this
-- migration is correct even against a database that already has rows.

ALTER TABLE role_resource_set_assignments ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE role_resource_set_assignments ADD CONSTRAINT role_resource_set_assignments_tenant_id_fkey
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
CREATE INDEX idx_role_resource_set_assignments_tenant ON role_resource_set_assignments(tenant_id);

-- --- Step 3: recreate both foreign keys against the composite key ---

ALTER TABLE role_resource_set_assignments
    ADD CONSTRAINT role_resource_set_assignments_resource_set_id_fkey
    FOREIGN KEY (tenant_id, resource_set_id) REFERENCES resource_sets(tenant_id, id) ON DELETE CASCADE;

ALTER TABLE api_tokens
    ADD CONSTRAINT api_tokens_resource_set_id_fkey
    FOREIGN KEY (tenant_id, resource_set_id) REFERENCES resource_sets(tenant_id, id) ON DELETE SET NULL;
