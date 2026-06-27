# dipole_trust_provider

Manages a Trust Provider in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_trust_provider" "k8s" {
  name = "K8s Production"
  type = "kubernetes_service_account"
  config_json = jsonencode({
    oidc_endpoint = "https://oidc.eks.ap-south-1.amazonaws.com/id/EXAMPLE"
    match_rules = [
      { field = "kubernetes.io.namespace", value = "production" }
    ]
  })
}
```

## Argument Reference

- `name` (Required) — Display name.
- `type` (Required) — Provider type.
- `config_json` (Required) — Configuration as JSON.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.
- `tags` (Optional) — Key-value tags.

## Attribute Reference

- `id` — UUID identifier.
- `edge_sdk_client_id` — Auto-generated Edge SDK Client ID.
- `edge_sdk_audience` — Auto-generated Edge SDK Audience.
- `tags_all` — All tags including provider defaults.

## Import

```bash
terraform import dipole_trust_provider.k8s <uuid>
```
