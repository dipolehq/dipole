/**
 * Centralized platform URL generation. All display URLs shown to customers
 * in the admin console derive from these functions. No hardcoded URLs
 * anywhere else in the frontend.
 *
 * Pattern: tenant-specific subdomains use <tenant>-mcp.dipolehq.com (flattened,
 * no region in hostname). The API base URL is used for callback/redirect URLs.
 */

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string) || 'https://api.dipolehq.com'

export function apiBaseUrl(): string {
  return API_BASE
}

export function mcpAuthServerUrl(tenantSubdomain: string): string {
  return `https://${tenantSubdomain}-mcp.dipolehq.com`
}

export function identityIssuerUrl(tenantSubdomain: string): string {
  return `https://${tenantSubdomain}-id.dipolehq.com`
}

export function mcpGatewayUrl(tenantSubdomain: string): string {
  return `https://${tenantSubdomain}-mcpgateway.dipolehq.com`
}

export function samlEntityId(): string {
  return `${API_BASE}/saml/metadata`
}

export function samlAcsUrl(): string {
  return `${API_BASE}/saml/acs`
}

export function oidcCallbackUrl(): string {
  return `${API_BASE}/callback/oidc`
}

export function adminSsoCallbackUrl(): string {
  return `${API_BASE}/api/v1/auth/sso/callback`
}

export function jwksUrl(): string {
  return `${API_BASE}/.well-known/openid-configuration/jwks`
}
