/**
 * Centralized platform URL generation. All display URLs shown to customers
 * in the admin console derive from these functions. No hardcoded URLs
 * anywhere else in the frontend.
 *
 * Pattern: per-tenant services use a HYPHEN-separated single label —
 * <tenant>-mcp, <tenant>-id, <tenant>-mcpgateway. This mirrors
 * cloud/internal/platform/urls.go, which is the backend's definition of the
 * same pattern. THE TWO MUST STAY IN AGREEMENT: customers copy values from
 * this UI into their Identity Provider, and the backend sends those same
 * values as redirect_uri. A divergence surfaces to the customer as
 * `redirect_uri_mismatch` from their IdP, which is painful to diagnose.
 *
 * Anything tenant-specific takes the tenant subdomain, because one console
 * serves every tenant. Use currentTenantSubdomain() to read the signed-in
 * tenant's own subdomain.
 */

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string) || 'https://api.dipolehq.com'

const PLATFORM_DOMAIN = (import.meta.env.VITE_PLATFORM_DOMAIN as string) || 'dipolehq.com'

/** Dipole's AWS Account ID used in cross-account bucket policies for log streaming. */
const DIPOLE_AWS_ACCOUNT_ID = (import.meta.env.VITE_DIPOLE_AWS_ACCOUNT_ID as string) || '992382430086'

/** Session storage key holding the signed-in tenant's subdomain. */
export const TENANT_SUBDOMAIN_KEY = 'dipole_tenant_subdomain'

// Mirrors platform.OIDCCallbackPath / platform.SAMLCallbackPath in
// cloud/internal/platform/urls.go. The backend registers its routes from those
// constants and builds the redirect_uri it sends the IdP from them too, so
// these must stay identical — a customer registers what we show here.
const OIDC_CALLBACK_PATH = '/callback/oidc'
const SAML_CALLBACK_PATH = '/callback/saml'

export function apiBaseUrl(): string {
  return API_BASE
}

export function dipoleAwsAccountId(): string {
  return DIPOLE_AWS_ACCOUNT_ID
}

/**
 * The signed-in tenant's subdomain.
 *
 * Read from the `tenant_subdomain` claim on the session token, which is the
 * authoritative source: it's issued together with the tenant scope of the
 * session, so it can't drift out of sync, and it survives a page reload.
 *
 * Returns an empty string when unknown (a session issued before the claim
 * existed, or an admin-SSO login — see later.md "Admin Console SSO Login Is
 * Single-Tenant"). Callers MUST treat empty as "unknown" and show nothing
 * rather than building a URL from it: `https://-mcp.dipolehq.com/...` looks
 * plausible enough that a customer would paste it into their IdP.
 */
export function currentTenantSubdomain(): string {
  const fromToken = tenantSubdomainFromSessionToken()
  if (fromToken) return fromToken

  // Transitional fallback for sessions issued before the claim was added.
  // Self-heals on the next login (session tokens are short-lived).
  return sessionStorage.getItem(TENANT_SUBDOMAIN_KEY) ?? ''
}

export function setCurrentTenantSubdomain(subdomain: string): void {
  if (subdomain) sessionStorage.setItem(TENANT_SUBDOMAIN_KEY, subdomain)
}

function tenantSubdomainFromSessionToken(): string {
  try {
    const token = sessionStorage.getItem('dipole_token')
    if (!token) return ''
    const segment = token.split('.')[1]
    if (!segment) return ''
    const payload = JSON.parse(atob(segment.replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload.tenant_subdomain === 'string' ? payload.tenant_subdomain : ''
  } catch {
    return ''
  }
}

/**
 * Renders a platform URL for display, or an explicit message when the tenant
 * isn't known. Never returns a half-built URL.
 */
export function displayUrl(url: string): string {
  return url || 'Unavailable — sign out and sign back in'
}

// ---------------------------------------------------------------------------
// Per-tenant service hosts. Keep in sync with platform.Hosts in Go.
// ---------------------------------------------------------------------------

// Every builder returns '' for an unknown tenant rather than a malformed
// hostname like `https://-mcp.dipolehq.com`. These values get copied into
// customers' IdP configuration, so a plausible-looking wrong URL is worse than
// no URL — it fails later, at the IdP, with an opaque error.

export function mcpAuthServerUrl(tenantSubdomain: string): string {
  if (!tenantSubdomain) return ''
  return `https://${tenantSubdomain}-mcp.${PLATFORM_DOMAIN}`
}

export function identityIssuerUrl(tenantSubdomain: string): string {
  if (!tenantSubdomain) return ''
  return `https://${tenantSubdomain}-id.${PLATFORM_DOMAIN}`
}

export function mcpGatewayUrl(tenantSubdomain: string): string {
  if (!tenantSubdomain) return ''
  return `https://${tenantSubdomain}-mcpgateway.${PLATFORM_DOMAIN}`
}

// ---------------------------------------------------------------------------
// Values the customer registers with their Identity Provider. These MUST match
// what the backend sends — see the note at the top of this file.
// ---------------------------------------------------------------------------

/**
 * Redirect URI for the MCP authorization flow. The backend builds this from the
 * hostname the authorize request arrived on (platform.OIDCCallbackPath on the
 * tenant's MCP Authorization Server host), so it is per tenant — NOT the shared
 * API host.
 */
export function oidcCallbackUrl(tenantSubdomain: string): string {
  const base = mcpAuthServerUrl(tenantSubdomain)
  return base ? `${base}${OIDC_CALLBACK_PATH}` : ''
}

/**
 * SAML Assertion Consumer Service URL. Points at the route the backend
 * actually serves (POST /callback/saml on the MCP Authorization Server host).
 */
export function samlAcsUrl(tenantSubdomain: string): string {
  const base = mcpAuthServerUrl(tenantSubdomain)
  return base ? `${base}${SAML_CALLBACK_PATH}` : ''
}

/**
 * SAML Service Provider entity ID. An identifier rather than a fetchable
 * endpoint; the tenant's MCP Authorization Server URL is used so each tenant
 * presents a distinct SP identity to the IdP.
 */
export function samlEntityId(tenantSubdomain: string): string {
  return mcpAuthServerUrl(tenantSubdomain)
}

/**
 * JWKS URL. Matches the `jwks_uri` our own OAuth metadata advertises, which is
 * on the MCP Authorization Server host.
 */
export function jwksUrl(tenantSubdomain: string): string {
  const base = mcpAuthServerUrl(tenantSubdomain)
  return base ? `${base}/.well-known/openid-configuration/jwks` : ''
}

// ---------------------------------------------------------------------------
// Console-level URLs (not tenant-specific — the console is one hostname).
// ---------------------------------------------------------------------------

/** Admin SSO callback, handled by the console's own auth routes. */
export function adminSsoCallbackUrl(): string {
  return `${API_BASE}/api/v1/auth/sso/callback`
}
