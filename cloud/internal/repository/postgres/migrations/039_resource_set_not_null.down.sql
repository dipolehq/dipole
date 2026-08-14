-- Revert: make resource_set_id nullable again and remove defaults.
ALTER TABLE client_workloads ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE server_workloads ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE trust_providers ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE credential_providers ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE access_conditions ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE access_policies ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE agent_controllers ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE agent_proxies ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE credential_provider_integrations ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE access_authorization_events ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
ALTER TABLE workload_events ALTER COLUMN resource_set_id DROP NOT NULL, ALTER COLUMN resource_set_id DROP DEFAULT;
