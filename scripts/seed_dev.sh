#!/bin/bash
set -e

DB_URL="${DIPOLE_DATABASE_URL:-postgres://dipole:devpassword@localhost:5432/dipole?sslmode=disable}"

echo "Seeding development data..."

psql "$DB_URL" <<'SQL'
INSERT INTO tenants (id, name, subdomain)
VALUES ('a0000000-0000-0000-0000-000000000001', 'Dev Tenant', 'dev')
ON CONFLICT (subdomain) DO NOTHING;

INSERT INTO client_workloads (id, tenant_id, name, description, identifiers, tags)
VALUES (
  'b0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Test App',
  'Development test workload',
  '[{"type": "k8s_namespace", "value": "default"}, {"type": "k8s_pod_name_prefix", "value": "testapp-"}]',
  '[{"key": "env", "value": "dev"}]'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO server_workloads (id, tenant_id, name, description, host, port, app_protocol, auth_method, auth_scheme)
VALUES (
  'c0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'HTTPBin',
  'Test HTTP endpoint',
  'httpbin.org',
  443,
  'HTTP',
  'http_bearer',
  'Bearer'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO credential_providers (id, tenant_id, name, description, is_active, type, config_encrypted)
VALUES (
  'd0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Test API Key',
  'Development API key for testing',
  true,
  'api_key',
  '\x00'::bytea
) ON CONFLICT (id) DO NOTHING;

INSERT INTO access_policies (id, tenant_id, name, is_active, client_workload_id, server_workload_id)
VALUES (
  'e0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Allow TestApp to HTTPBin',
  true,
  'b0000000-0000-0000-0000-000000000001',
  'c0000000-0000-0000-0000-000000000001'
) ON CONFLICT (id) DO NOTHING;

INSERT INTO policy_credential_providers (policy_id, credential_provider_id, mapping_type, mapping_value)
VALUES (
  'e0000000-0000-0000-0000-000000000001',
  'd0000000-0000-0000-0000-000000000001',
  'None',
  ''
) ON CONFLICT (policy_id, credential_provider_id) DO NOTHING;
SQL

echo "Seed complete."
