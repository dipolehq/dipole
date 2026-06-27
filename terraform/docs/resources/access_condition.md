# dipole_access_condition

Manages an Access Condition in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_access_condition" "business_hours" {
  name = "Business Hours Only"
  type = "time_condition"
  config_json = jsonencode({
    days       = ["Mon", "Tue", "Wed", "Thu", "Fri"]
    start_time = "09:00"
    end_time   = "18:00"
    timezone   = "Asia/Kolkata"
  })
}
```

## Argument Reference

- `name` (Required) — Display name.
- `type` (Required) — Condition type (time_condition, geoip_condition).
- `config_json` (Required) — Configuration as JSON.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.
- `tags` (Optional) — Key-value tags.

## Attribute Reference

- `id` — UUID identifier.
- `tags_all` — All tags including provider defaults.

## Import

```bash
terraform import dipole_access_condition.business_hours <uuid>
```
