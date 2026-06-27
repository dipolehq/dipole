# dipole_credential_provider

Manages a Credential Provider in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_credential_provider" "stripe_key" {
  name = "Stripe API Key"
  type = "api_key"
  config_json = jsonencode({
    api_key = var.stripe_api_key
  })
}
```

## Argument Reference

- `name` (Required) — Display name.
- `type` (Required) — Credential type.
- `config_json` (Required, Sensitive) — Configuration as JSON. Contains secrets.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.
- `tags` (Optional) — Key-value tags.

## Attribute Reference

- `id` — UUID identifier.
- `tags_all` — All tags including provider defaults.

## Import

```bash
terraform import dipole_credential_provider.stripe_key <uuid>
```

Note: After import, `config_json` will be empty in state. Run `terraform apply` to sync.
