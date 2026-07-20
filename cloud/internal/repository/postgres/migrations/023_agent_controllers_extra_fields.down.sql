ALTER TABLE agent_controllers DROP COLUMN IF EXISTS description;
ALTER TABLE agent_controllers DROP COLUMN IF EXISTS tls_hostname;
ALTER TABLE agent_controllers DROP COLUMN IF EXISTS tags;
ALTER TABLE agent_controllers DROP COLUMN IF EXISTS is_active;
ALTER TABLE agent_controllers DROP COLUMN IF EXISTS updated_at;
