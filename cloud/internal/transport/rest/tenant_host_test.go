package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
)

type fakeTenantLookup struct {
	bySubdomain map[string]*domain.Tenant
	calls       int
	err         error
}

func (f *fakeTenantLookup) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.bySubdomain[subdomain], nil
}

func newTestResolver(allowHeader bool) (*TenantHostResolver, *fakeTenantLookup) {
	tenant := &domain.Tenant{ID: uuid.New(), Name: "Acme", Subdomain: "647251"}
	lookup := &fakeTenantLookup{bySubdomain: map[string]*domain.Tenant{"647251": tenant}}
	return NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), allowHeader), lookup
}

func requestWith(host, forwardedHost string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Host = host
	if forwardedHost != "" {
		req.Header.Set("X-Forwarded-Host", forwardedHost)
	}
	return req
}

func TestResolve_FromForwardedHost(t *testing.T) {
	r, _ := newTestResolver(false)

	got, err := r.Resolve(context.Background(), requestWith("internal.run.app", "647251-mcp.dipolehq.com"))
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got.Subdomain != "647251" {
		t.Errorf("Subdomain = %q, want 647251", got.Subdomain)
	}
	if got.Service != platform.ServiceMCPAuth {
		t.Errorf("Service = %q, want mcp", got.Service)
	}
	if want := "https://647251-mcp.dipolehq.com"; got.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, want)
	}
}

func TestResolve_ForwardedHostTakesPrecedenceOverHost(t *testing.T) {
	r, _ := newTestResolver(false)

	_, err := r.Resolve(context.Background(), requestWith("dipole-cloud-abc-uc.a.run.app", "647251-mcp.dipolehq.com"))
	if err != nil {
		t.Fatalf("expected forwarded host to be used, got error: %v", err)
	}
}

func TestResolve_ForwardedHostChainUsesFirstEntry(t *testing.T) {
	r, _ := newTestResolver(false)

	got, err := r.Resolve(context.Background(), requestWith("x", "647251-mcp.dipolehq.com, edge.internal"))
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got.Subdomain != "647251" {
		t.Errorf("Subdomain = %q, want 647251", got.Subdomain)
	}
}

func TestResolve_FallsBackToHostWhenNoForwardedHeader(t *testing.T) {
	r, _ := newTestResolver(false)

	got, err := r.Resolve(context.Background(), requestWith("647251-mcp.dipolehq.com", ""))
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got.Subdomain != "647251" {
		t.Errorf("Subdomain = %q, want 647251", got.Subdomain)
	}
}

func TestResolve_IdentityHostResolvesIdentityService(t *testing.T) {
	r, _ := newTestResolver(false)

	got, err := r.Resolve(context.Background(), requestWith("x", "647251-id.dipolehq.com"))
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got.Service != platform.ServiceIdentity {
		t.Errorf("Service = %q, want id", got.Service)
	}
	if want := "https://647251-id.dipolehq.com"; got.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, want)
	}
}

func TestResolve_UnresolvableHostFailsClosed(t *testing.T) {
	r, _ := newTestResolver(false)

	for _, host := range []string{"api.dipolehq.com", "console.dipolehq.com", "dipolehq.com", "647251-mcp.evil.com"} {
		if _, err := r.Resolve(context.Background(), requestWith("x", host)); !errors.Is(err, ErrTenantHostUnresolved) {
			t.Errorf("Resolve(%q) error = %v, want ErrTenantHostUnresolved", host, err)
		}
	}
}

func TestResolve_UnknownTenantFailsClosed(t *testing.T) {
	r, _ := newTestResolver(false)

	if _, err := r.Resolve(context.Background(), requestWith("x", "999999-mcp.dipolehq.com")); !errors.Is(err, ErrTenantNotFound) {
		t.Errorf("error = %v, want ErrTenantNotFound", err)
	}
}

func TestResolve_DevHeaderIgnoredWhenDisabled(t *testing.T) {
	r, _ := newTestResolver(false)

	req := requestWith("x", "api.dipolehq.com")
	req.Header.Set(DevTenantHeader, "647251")

	if _, err := r.Resolve(context.Background(), req); err == nil {
		t.Fatal("dev tenant header was honoured with override disabled")
	}
}

func TestResolve_DevHeaderWorksWhenEnabled(t *testing.T) {
	r, _ := newTestResolver(true)

	req := requestWith("localhost:8080", "")
	req.Header.Set(DevTenantHeader, "647251")

	got, err := r.Resolve(context.Background(), req)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got.Subdomain != "647251" {
		t.Errorf("Subdomain = %q, want 647251", got.Subdomain)
	}
	if want := "https://647251-mcp.dipolehq.com"; got.BaseURL != want {
		t.Errorf("BaseURL = %q, want %q", got.BaseURL, want)
	}
}

func TestResolve_CachesPositiveResult(t *testing.T) {
	r, lookup := newTestResolver(false)

	for i := 0; i < 5; i++ {
		if _, err := r.Resolve(context.Background(), requestWith("x", "647251-mcp.dipolehq.com")); err != nil {
			t.Fatalf("Resolve error: %v", err)
		}
	}
	if lookup.calls != 1 {
		t.Errorf("DB lookups = %d, want 1 (result should be cached)", lookup.calls)
	}
}

func TestResolve_CachesNegativeResult(t *testing.T) {
	r, lookup := newTestResolver(false)

	for i := 0; i < 5; i++ {
		_, _ = r.Resolve(context.Background(), requestWith("x", "999999-mcp.dipolehq.com"))
	}
	if lookup.calls != 1 {
		t.Errorf("DB lookups = %d, want 1 (negative result should be cached)", lookup.calls)
	}
}

func TestResolve_DoesNotCacheLookupErrors(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "647251"}
	lookup := &fakeTenantLookup{
		bySubdomain: map[string]*domain.Tenant{"647251": tenant},
		err:         errors.New("connection refused"),
	}
	r := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	if _, err := r.Resolve(context.Background(), requestWith("x", "647251-mcp.dipolehq.com")); err == nil {
		t.Fatal("expected error from failing lookup")
	}

	lookup.err = nil
	if _, err := r.Resolve(context.Background(), requestWith("x", "647251-mcp.dipolehq.com")); err != nil {
		t.Fatalf("Resolve after recovery: %v", err)
	}
	if lookup.calls != 2 {
		t.Errorf("DB lookups = %d, want 2 (error must not be cached)", lookup.calls)
	}
}

func TestResolveTenantID(t *testing.T) {
	r, _ := newTestResolver(false)

	id, err := r.ResolveTenantID(context.Background(), requestWith("x", "647251-mcp.dipolehq.com"))
	if err != nil {
		t.Fatalf("ResolveTenantID error: %v", err)
	}
	if id == uuid.Nil {
		t.Error("ResolveTenantID returned uuid.Nil")
	}

	if _, err := r.ResolveTenantID(context.Background(), requestWith("x", "api.dipolehq.com")); err == nil {
		t.Error("expected error for non-tenant host")
	}
}

func TestPublicHost(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		forwarded string
		want      string
	}{
		{"forwarded wins", "internal.run.app", "647251-mcp.dipolehq.com", "647251-mcp.dipolehq.com"},
		{"falls back to host", "647251-mcp.dipolehq.com", "", "647251-mcp.dipolehq.com"},
		{"first of chain", "internal", "a.dipolehq.com, b.internal", "a.dipolehq.com"},
		{"whitespace trimmed", "internal", "  a.dipolehq.com  ", "a.dipolehq.com"},
		{"blank forwarded falls back", "647251-mcp.dipolehq.com", "   ", "647251-mcp.dipolehq.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PublicHost(requestWith(tt.host, tt.forwarded)); got != tt.want {
				t.Errorf("PublicHost = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolve_ConcurrentAccessIsRaceFree(t *testing.T) {
	tenantA := &domain.Tenant{ID: uuid.New(), Subdomain: "111111"}
	tenantB := &domain.Tenant{ID: uuid.New(), Subdomain: "222222"}
	lookup := &concurrentTenantLookup{bySubdomain: map[string]*domain.Tenant{
		"111111": tenantA,
		"222222": tenantB,
	}}
	r := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	hosts := []string{
		"111111-mcp.dipolehq.com",
		"111111-id.dipolehq.com",
		"222222-mcp.dipolehq.com",
		"999999-mcp.dipolehq.com",
		"api.dipolehq.com",
	}

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			host := hosts[i%len(hosts)]
			for j := 0; j < 20; j++ {
				resolved, err := r.Resolve(context.Background(), requestWith("x", host))
				if err == nil && resolved != nil {
					if !strings.HasPrefix(host, resolved.Subdomain+"-") {
						t.Errorf("host %q resolved to subdomain %q", host, resolved.Subdomain)
						return
					}
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestResolve_CacheEvictionUnderConcurrency(t *testing.T) {
	lookup := &concurrentTenantLookup{bySubdomain: map[string]*domain.Tenant{}}
	r := NewTenantHostResolver(lookup, platform.NewHosts("dipolehq.com", ""), false)

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 2000; i++ {
				host := fmt.Sprintf("t%d%d-mcp.dipolehq.com", g, i)
				_, _ = r.Resolve(context.Background(), requestWith("x", host))
			}
		}(g)
	}
	wg.Wait()

	r.mu.RLock()
	size := len(r.cache)
	r.mu.RUnlock()
	if size > 10001 {
		t.Errorf("cache grew to %d entries despite the bound", size)
	}
}

type concurrentTenantLookup struct {
	bySubdomain map[string]*domain.Tenant
}

func (c *concurrentTenantLookup) GetBySubdomain(_ context.Context, subdomain string) (*domain.Tenant, error) {
	return c.bySubdomain[subdomain], nil
}
