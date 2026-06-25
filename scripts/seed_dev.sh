#!/bin/bash
# seed_dev.sh — inserts a complete test policy for local development.
# Run after docker-compose up: ./scripts/seed_dev.sh
#
# Creates: tenant → client workload → server workload → credential provider → access policy
# The policy allows any pod in namespace "default" to access httpbin.org:443 with a test API key.

set -e

DB_URL="${DIPOLE_DATABASE_URL:-postgres://dipole:devpassword@localhost:5432/dipole?sslmode=disable}"
ENC_KEY_HEX="0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

echo "Seeding development data..."

# Generate encrypted credential config using Go
ENCRYPTED_HEX=$(cd /tmp && cat > encrypt_seed.go << 'GOEOF'
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func main() {
	keyHex := os.Args[1]
	plaintext := os.Args[2]

	key, _ := hex.DecodeString(keyHex)
	block, _ := aes.NewCipher(key)
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	fmt.Print(hex.EncodeToString(ciphertext))
}
GOEOF
go run encrypt_seed.go "$ENC_KEY_HEX" '{"api_key":"test-policy-credential-xyz"}')

echo "Encrypted credential config: ${ENCRYPTED_HEX:0:20}..."

psql "$DB_URL" << SQL
-- Tenant
INSERT INTO tenants (id, name, subdomain)
VALUES ('a0000000-0000-0000-0000-000000000001', 'Dev Tenant', 'dev')
ON CONFLICT (id) DO NOTHING;

-- Client Workload: matches namespace=default
INSERT INTO client_workloads (id, tenant_id, name, identifiers)
VALUES (
  'b0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Test Client',
  '[{"type": "k8s_namespace", "value": "default"}]'
)
ON CONFLICT (id) DO NOTHING;

-- Server Workload: httpbin.org:443 with Bearer auth
INSERT INTO server_workloads (id, tenant_id, name, host, port, app_protocol, auth_method, auth_scheme)
VALUES (
  'c0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'HTTPBin',
  'httpbin.org',
  443,
  'HTTP',
  'http_bearer',
  'Bearer'
)
ON CONFLICT (id) DO NOTHING;

-- Credential Provider: encrypted API key
INSERT INTO credential_providers (id, tenant_id, name, type, config_encrypted)
VALUES (
  'd0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Test Key',
  'api_key',
  decode('$ENCRYPTED_HEX', 'hex')
)
ON CONFLICT (id) DO NOTHING;

-- Access Policy: links client → server (active)
INSERT INTO access_policies (id, tenant_id, name, is_active, client_workload_id, server_workload_id)
VALUES (
  'e0000000-0000-0000-0000-000000000001',
  'a0000000-0000-0000-0000-000000000001',
  'Test Policy',
  true,
  'b0000000-0000-0000-0000-000000000001',
  'c0000000-0000-0000-0000-000000000001'
)
ON CONFLICT (id) DO NOTHING;

-- Link policy to credential provider
INSERT INTO policy_credential_providers (policy_id, credential_provider_id, mapping_type, mapping_value)
VALUES (
  'e0000000-0000-0000-0000-000000000001',
  'd0000000-0000-0000-0000-000000000001',
  'None',
  ''
)
ON CONFLICT (policy_id, credential_provider_id) DO NOTHING;
SQL

echo ""
echo "Seed complete. Policy: default namespace → httpbin.org:443 with Bearer test-policy-credential-xyz"
echo ""
echo "Test with:"
echo "  HTTPS_PROXY=http://localhost:8000 curl -sk https://httpbin.org/headers"
