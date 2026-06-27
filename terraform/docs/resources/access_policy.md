# dipole_access_policy

Manages an Access Policy in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_access_policy" "app_to_stripe" {
  name                   = "My App to Stripe"
  is_active              = true
  client_workload_id     = dipole_client_workload.my_app.id
  server_workload_id     = dipole_server_workload.stripe.id
  trust_provider_ids     = [dipole_trust_provider.k8s.id]
  credential_provider_id = dipole_credential_provider.stripe_key.id
}
```

## Argument Reference

- `name` (Required) — Display name.
- `client_workload_id` (Required) — Client Workload ID.
- `server_workload_id` (Required) — Server Workload ID.
- `trust_provider_ids` (Optional, Set of String) — Trust Provider IDs.
- `credential_provider_id` (Optional) — Credential Provider ID (simple case).
- `credential_providers` (Optional, Set) — Multiple credential providers with mapping.
- `access_condition_ids` (Optional, Set of String) — Access Condition IDs.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.

## Attribute Reference

- `id` — UUID identifier.

## Import

```bash
terraform import dipole_access_policy.app_to_stripe <uuid>
```
