# dipole_server_workload

Manages a Server Workload in a Dipole tenant.

## Example Usage

```hcl
resource "dipole_server_workload" "stripe" {
  name         = "Stripe API"
  host         = "api.stripe.com"
  port         = 443
  app_protocol = "HTTP"
  tls          = true
  auth_method  = "http_bearer"
}
```

## Argument Reference

- `name` (Required) — Display name.
- `host` (Required) — Service endpoint hostname. Wildcards supported.
- `app_protocol` (Required) — HTTP, PostgreSQL, MySQL, Redis, Snowflake, etc.
- `port` (Optional, default 443) — Port.
- `tls` (Optional, default true) — TLS enabled.
- `tls_verification` (Optional, default "full") — full or none.
- `transport_protocol` (Optional, default "TCP") — Transport protocol.
- `requested_port` (Optional) — Forward-to port.
- `requested_tls` (Optional) — Proxy-to-server TLS.
- `auth_method` (Optional) — Authentication method.
- `auth_scheme` (Optional) — Authentication scheme.
- `auth_config` (Optional, Map) — Additional auth configuration.
- `http_headers` (Optional, Map) — Static HTTP headers.
- `url_path` (Optional) — URL path.
- `description` (Optional) — Description.
- `is_active` (Optional, default true) — Active status.
- `resource_set_id` (Optional) — Resource Set ID.
- `tags` (Optional) — Key-value tags.

## Attribute Reference

- `id` — UUID identifier.
- `tags_all` — All tags including provider defaults.

## Import

```bash
terraform import dipole_server_workload.stripe <uuid>
```
