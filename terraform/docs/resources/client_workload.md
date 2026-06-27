# dipole_client_workload

Manages a Client Workload in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_client_workload" "my_app" {
  name = "My Application"
  identifiers = [
    { type = "k8s_namespace", value = "production" },
    { type = "k8s_pod_name_prefix", value = "myapp-" }
  ]
  tags = { team = "platform" }
}
```

## Argument Reference

- `name` (Required) — Display name.
- `identifiers` (Required, Set) — Identity attributes (type + value). AND logic.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.
- `tags` (Optional) — Key-value tags.

## Attribute Reference

- `id` — UUID identifier.
- `tags_all` — All tags including provider defaults.

## Import

```bash
terraform import dipole_client_workload.my_app <uuid>
```
