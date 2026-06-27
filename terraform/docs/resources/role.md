# dipole_role

Manages a Role in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_role" "developer" {
  name = "Developer"
  permissions_json = jsonencode({
    client_workloads     = { read = true, write = true }
    server_workloads     = { read = true, write = true }
    trust_providers      = { read = true, write = false }
    credential_providers = { read = true, write = false }
    access_policies      = { read = true, write = true }
    access_conditions    = { read = true, write = false }
    agent_controllers    = { read = true, write = false }
    users                = { read = false, write = false }
    roles                = { read = false, write = false }
    resource_sets        = { read = true, write = false }
    log_streams          = { read = false, write = false }
    identity_providers   = { read = false, write = false }
  })
}
```

## Argument Reference

- `name` (Required) — Display name.
- `permissions_json` (Required) — RBAC permissions as JSON.

## Attribute Reference

- `id` — UUID identifier.

## Import

```bash
terraform import dipole_role.developer <uuid>
```
