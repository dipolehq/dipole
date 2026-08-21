// tenant_host.go resolves which tenant a request belongs to from its public hostname.
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
	ErrTenantHostUnresolved   = errors.New("request hostname does not identify a tenant")
	ErrTenantNotFound         = errors.New("no tenant matches the request hostname")
	ErrTenantSubdomainInvalid = errors.New("tenant subdomain is not a valid hostname label")
)

type TenantLookup interface {
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error)
}

type ResolvedTenantHost struct {
	Tenant    *domain.Tenant
	Subdomain string
	Service   platform.Service
	BaseURL   string
}

func (r *ResolvedTenantHost) TenantID() uuid.UUID { return r.Tenant.ID }

type tenantCacheEntry struct {
	tenant    *domain.Tenant
	expiresAt time.Time
}

type TenantHostResolver struct {
	tenants             TenantLookup
	hosts               *platform.Hosts
	allowHeaderOverride bool
	ttl                 time.Duration
	negativeTTL         time.Duration

	mu    sync.RWMutex
	cache map[string]tenantCacheEntry
}

const (
	defaultTenantCacheTTL         = 60 * time.Second
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

func (r *TenantHostResolver) Resolve(ctx context.Context, req *http.Request) (*ResolvedTenantHost, error) {
	subdomain, svc, _, ok := r.hosts.ParseHost(PublicHost(req))
	if !ok {
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

func (r *TenantHostResolver) ResolveTenantID(ctx context.Context, req *http.Request) (uuid.UUID, error) {
	resolved, err := r.Resolve(ctx, req)
	if err != nil {
		return uuid.Nil, err
	}
	return resolved.Tenant.ID, nil
}

func (r *TenantHostResolver) BaseURLFor(tenantSubdomain string, svc platform.Service) string {
	return r.hosts.URLForService(tenantSubdomain, svc)
}

const DevTenantHeader = "X-Dipole-Tenant-Subdomain"

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

	if len(r.cache) > 10000 {
		r.cache = make(map[string]tenantCacheEntry)
	}
	r.cache[key] = tenantCacheEntry{tenant: tenant, expiresAt: time.Now().Add(ttl)}
}

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
