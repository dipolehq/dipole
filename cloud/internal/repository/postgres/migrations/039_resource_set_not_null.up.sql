-- Migration 039: Enforce resource_set_id NOT NULL on all resource-scoped tables.
--
-- Every resource-scoped entity must belong to exactly one resource set.
-- Entities without an explicit assignment are backfilled into the Default
-- resource set (ffffffff-ffff-ffff-ffff-ffffffffffff).
--
-- This prevents future NULLs and guarantees that resource set filtering
-- in queries always produces deterministic results without needing
-- OR resource_set_id IS NULL fallback clauses.

-- Step 1: Backfill NULL rows to Default resource set.
UPDATE client_workloads
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE server_workloads
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE trust_providers
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE credential_providers
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE access_conditions
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE access_policies
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE agent_controllers
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE agent_proxies
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE credential_provider_integrations
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE access_authorization_events
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

UPDATE workload_events
SET resource_set_id = 'ffffffff-ffff-ffff-ffff-ffffffffffff'
WHERE resource_set_id IS NULL;

-- Step 2: Add NOT NULL constraint (prevents future NULL inserts).
ALTER TABLE client_workloads
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE server_workloads
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE trust_providers
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE credential_providers
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE access_conditions
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE access_policies
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE agent_controllers
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE agent_proxies
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE credential_provider_integrations
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

-- Partitioned tables: ALTER on the parent cascades to all partitions.
ALTER TABLE access_authorization_events
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';

ALTER TABLE workload_events
ALTER COLUMN resource_set_id SET NOT NULL,
ALTER COLUMN resource_set_id SET DEFAULT 'ffffffff-ffff-ffff-ffff-ffffffffffff';
