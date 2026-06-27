# Dipole Provider

The Dipole provider manages workload IAM resources as infrastructure-as-code.

## Example Usage

```hcl
provider "dipole" {
  tenant_id = "acme"
  token     = var.dipole_api_token
  region    = "ap-south-1"
}
```

## Authentication

Two methods:
- **Native OIDC** (recommended for CI/CD): Set `client_id` from a Trust Provider.
- **Direct token** (dev/testing): Set `tenant_id` + `token`.

All values can be set via environment variables: `DIPOLE_CLIENT_ID`, `DIPOLE_TENANT_ID`, `DIPOLE_API_TOKEN`, `DIPOLE_REGION`, `DIPOLE_RESOURCE_SET_ID`.

## Schema

### Optional

- `client_id` (String) — Trust Provider Client ID for native OIDC auth.
- `tenant_id` (String) — Tenant subdomain.
- `token` (String, Sensitive) — API token.
- `region` (String) — Region (default: ap-south-1).
- `resource_set_id` (String) — Default Resource Set for all resources.
- `default_tags` (Map of String) — Tags applied to all resources.
