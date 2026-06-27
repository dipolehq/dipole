# dipole_resource_set

Manages a Resource Set in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_resource_set" "staging" {
  name        = "Staging Environment"
  description = "All staging workloads"
}
```

## Argument Reference

- `name` (Required) — Display name.
- `description` (Optional) — Description.

## Attribute Reference

- `id` — UUID identifier.

## Import

```bash
terraform import dipole_resource_set.staging <uuid>
```
