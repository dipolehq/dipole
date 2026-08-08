/*
Package rest — tenant_host.go resolves which tenant a request belongs to from
its public hostname.

This is the single mechanism for hostname-based tenant resolution. It exists
because the Cloud process serves EVERY tenant: binding a tenant at startup
(from an env var) silently served every request from one tenant regardless of
which hostname it arrived on.

Two rules this file enforces:

 1. Read X-Forwarded-Host, NOT r.Host. The Cloudflare Worker in front of Cloud
    Run rewrites Host to Cloud Run's internal *.run.app hostname; only
    X-Forwarded-Host carries the hostname the client actually used.

 2. Fail closed. If the tenant cannot be determined, return an error. There is
    deliberately no "default tenant" fallback — that fallback is exactly what
    caused requests to be served from the wrong tenant.

SECURITY PREREQUISITE: because tenant identity is derived from a request
header, Cloud Run's raw *.run.app URL must NOT be publicly reachable, or a
caller could forge X-Forwarded-Host and act as any tenant. Restrict Cloud Run
ingress to internal-and-cloud-load-balancing, or have the Worker attach a
shared secret that this service verifies.
*/
package rest

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
)

var (
	// ErrTenantHostUnresolved means the hostname isn't a per-tenant service
	// host we serve (wrong domain, unknown service, apex, etc.).
	ErrTenantHostUnresolved = errors.New("request hostname does not identify a tenant")
	// ErrTenantNotFound means the hostname was well-formed but no such tenant exists.
	ErrTenantNotFound = errors.New("no tenant matches the request hostname")
	// ErrTenantSubdomainInvalid means a tenant was found, but its stored
	// subdomain can't form a valid hostname — so we cannot build the canonical
	// service URLs this request needs.
	ErrTenantSubdomainInvalid = errors.New("tenant subdomain is not a valid hostname label")
)

// TenantLookup is the subset of the tenant repository this resolver needs.
type TenantLookup interface {
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error)
}

// ResolvedTenantHost is the outcome of resolving one request.
type ResolvedTenantHost struct {
	Tenant    *domain.Tenant
	Subdomain string
	Service   platform.Service
	// BaseURL is the canonical URL of the service this request addressed, e.g.
	// https://647251-mcp.dipolehq.com. Always the canonical form (never the raw
	// request host) so that values we advertise in OIDC metadata or send as an
	// IdP redirect_uri are stable and match what the admin console shows.
	BaseURL string
}

func (r *ResolvedTenantHost) TenantID() uuid.UUID { return r.Tenant.ID }

type tenantCacheEntry struct {
	tenant    *domain.Tenant // nil = negative result
	expiresAt time.Time
}

// TenantHostResolver resolves tenants from request hostnames, with a short TTL
// cache so a per-request DB lookup doesn't sit in the hot path of every MCP
// tool call.
type TenantHostResolver struct {
	tenants TenantLookup
	hosts   *platform.Hosts

	// allowHeaderOverride permits selecting the tenant with the
	// DevTenantHeader. This is a LOCAL DEV ONLY affordance (there's no wildcard
	// DNS on a laptop) and must be false in any deployed environment: these
	// routes are unauthenticated, so with it enabled any caller could act as
	// any tenant.
	allowHeaderOverride bool

	ttl         time.Duration
	negativeTTL time.Duration

	mu    sync.RWMutex
	cache map[string]tenantCacheEntry
}

const (
	defaultTenantCacheTTL = 60 * time.Second
	// Negative results are cached only briefly: long enough to blunt repeated
	// lookups for a nonexistent subdomain, short enough that a tenant which
	// just signed up isn't rejected for a noticeable window.
	defaultTenantNegativeCacheTTL = 5 * time.Second
)

func NewTenantHostResolver(tenants TenantLookup, hosts *platform.Hosts, allowHeaderOverride bool) *TenantHostResolver {
	return &TenantHostResolver{
		tenants:             tenants,
		hosts:               hosts,
		allowHeaderOverride: allowHeaderOverride,
		ttl:                 defaultTenantCacheTTL,
		negativeTTL:         defaultTenantNegativeCacheTTL,
		cache:               make(map[string]tenantCacheEntry),
	}
}

// Resolve determines the tenant and the service host for a request.
// Callers must treat any error as fatal for the request (fail closed).
func (r *TenantHostResolver) Resolve(ctx context.Context, req *http.Request) (*ResolvedTenantHost, error) {
	subdomain, svc, ok := r.hosts.ParseHost(PublicHost(req))
	if !ok {
		// Local dev fallback only. Deployed environments have this disabled, so
		// an unresolvable hostname is a hard failure there.
		if r.allowHeaderOverride {
			if resolved, err := r.resolveFromHeader(ctx, req); err != nil || resolved != nil {
				return resolved, err
			}
		}
		return nil, ErrTenantHostUnresolved
	}

	tenant, err := r.lookup(ctx, subdomain)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// Single choke point for URL validity. Resolving successfully but with an
	// unbuildable BaseURL would let callers emit documents containing an empty
	// issuer (e.g. {"issuer": "", "token_endpoint": "/connect/token"}), so this
	// is treated as a resolution failure rather than pushed onto every caller.
	baseURL := r.hosts.URLForService(tenant.Subdomain, svc)
	if baseURL == "" {
		return nil, ErrTenantSubdomainInvalid
	}

	return &ResolvedTenantHost{
		Tenant:    tenant,
		Subdomain: tenant.Subdomain,
		Service:   svc,
		BaseURL:   baseURL,
	}, nil
}

// ResolveTenantID is a convenience for callers that only need the tenant UUID.
func (r *TenantHostResolver) ResolveTenantID(ctx context.Context, req *http.Request) (uuid.UUID, error) {
	resolved, err := r.Resolve(ctx, req)
	if err != nil {
		return uuid.Nil, err
	}
	return resolved.Tenant.ID, nil
}

// BaseURLFor builds the canonical URL for a service, for callers that already
// know the tenant subdomain (e.g. building a token's `iss` during policy
// evaluation, where there's no HTTP request to read).
func (r *TenantHostResolver) BaseURLFor(tenantSubdomain string, svc platform.Service) string {
	return r.hosts.URLForService(tenantSubdomain, svc)
}

// DevTenantHeader selects the tenant by subdomain in local dev, where there's
// no wildcard DNS to produce a real per-tenant hostname.
//
// It carries a SUBDOMAIN rather than a tenant UUID on purpose: it reuses the
// same GetBySubdomain lookup as the hostname path (so dev and prod exercise
// identical resolution logic), and a subdomain is what a developer actually has
// on hand from the console.
const DevTenantHeader = "X-Dipole-Tenant-Subdomain"

// resolveFromHeader implements the dev-only header path. Returns (nil, nil)
// when the header is absent so the caller falls through to failing closed.
func (r *TenantHostResolver) resolveFromHeader(ctx context.Context, req *http.Request) (*ResolvedTenantHost, error) {
	subdomain := strings.TrimSpace(req.Header.Get(DevTenantHeader))
	if subdomain == "" {
		return nil, nil
	}

	tenant, err := r.lookup(ctx, strings.ToLower(subdomain))
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		return nil, ErrTenantNotFound
	}

	// No service is encoded in the header, so assume the MCP Authorization
	// Server — the only component that consumed the old header.
	baseURL := r.hosts.URLForService(tenant.Subdomain, platform.ServiceMCPAuth)
	if baseURL == "" {
		return nil, ErrTenantSubdomainInvalid
	}

	return &ResolvedTenantHost{
		Tenant:    tenant,
		Subdomain: tenant.Subdomain,
		Service:   platform.ServiceMCPAuth,
		BaseURL:   baseURL,
	}, nil
}

func (r *TenantHostResolver) lookup(ctx context.Context, subdomain string) (*domain.Tenant, error) {
	if cached, ok := r.fromCache(subdomain); ok {
		return cached, nil
	}

	tenant, err := r.tenants.GetBySubdomain(ctx, subdomain)
	if err != nil {
		// Don't cache infrastructure errors — a transient DB blip shouldn't
		// pin a negative result for the whole TTL.
		return nil, err
	}

	r.store(subdomain, tenant)
	return tenant, nil
}

func (r *TenantHostResolver) fromCache(key string) (*domain.Tenant, bool) {
	r.mu.RLock()
	entry, ok := r.cache[key]
	r.mu.RUnlock()

	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.tenant, true
}

func (r *TenantHostResolver) store(key string, tenant *domain.Tenant) {
	ttl := r.ttl
	if tenant == nil {
		ttl = r.negativeTTL
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Bound the cache so an attacker probing random subdomains can't grow it
	// without limit. At this size a full reset is cheap and entries are
	// re-populated on demand.
	if len(r.cache) > 10000 {
		r.cache = make(map[string]tenantCacheEntry)
	}
	r.cache[key] = tenantCacheEntry{tenant: tenant, expiresAt: time.Now().Add(ttl)}
}

// PublicHost returns the hostname the client actually addressed.
//
// X-Forwarded-Host takes precedence over r.Host because the Cloudflare Worker
// rewrites Host to Cloud Run's internal *.run.app hostname when it proxies.
// When several proxies chain, the header is a comma-separated list whose first
// element is the original client-facing host.
func PublicHost(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		if i := strings.Index(fwd, ","); i >= 0 {
			fwd = fwd[:i]
		}
		if h := strings.TrimSpace(fwd); h != "" {
			return h
		}
	}
	return r.Host
}
