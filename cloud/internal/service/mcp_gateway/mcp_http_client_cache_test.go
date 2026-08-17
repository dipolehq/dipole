package mcp_gateway

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHTTPMCPClientFactory_ClientFor_ReusesClientForSameTenantAndServer(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	tenantID := uuid.New()
	server := ServerInfo{ID: uuid.New(), Host: "jira.example.com", Port: 443, TLS: true}

	c1 := factory.ClientFor(tenantID, server)
	c2 := factory.ClientFor(tenantID, server)

	if c1 != c2 {
		t.Error("expected the same cached client instance to be returned for repeated calls with the same tenant and server")
	}
}

func TestHTTPMCPClientFactory_ClientFor_IsolatesDifferentTenants(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	server := ServerInfo{ID: uuid.New(), Host: "jira.example.com", Port: 443, TLS: true}

	c1 := factory.ClientFor(uuid.New(), server)
	c2 := factory.ClientFor(uuid.New(), server)

	if c1 == c2 {
		t.Error("expected different tenants to receive different client instances even for the same server workload ID")
	}
}

func TestHTTPMCPClientFactory_ClientFor_IsolatesDifferentServers(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	tenantID := uuid.New()

	c1 := factory.ClientFor(tenantID, ServerInfo{ID: uuid.New(), Host: "jira.example.com", Port: 443, TLS: true})
	c2 := factory.ClientFor(tenantID, ServerInfo{ID: uuid.New(), Host: "slack.example.com", Port: 443, TLS: true})

	if c1 == c2 {
		t.Error("expected different server workloads to receive different client instances")
	}
}

func TestHTTPMCPClientFactory_ClientFor_InvalidatesOnBaseURLChange(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	tenantID := uuid.New()
	serverID := uuid.New()

	original := factory.ClientFor(tenantID, ServerInfo{ID: serverID, Host: "old-host.example.com", Port: 443, TLS: true})

	updated := factory.ClientFor(tenantID, ServerInfo{ID: serverID, Host: "new-host.example.com", Port: 443, TLS: true})

	if original == updated {
		t.Fatal("expected a changed base URL to invalidate the cached client, got the same instance back")
	}

	httpUpdated, ok := updated.(*HTTPMCPClient)
	if !ok {
		t.Fatal("expected *HTTPMCPClient")
	}
	if httpUpdated.baseURL != "https://new-host.example.com" {
		t.Errorf("expected updated client to use the new base URL, got %q", httpUpdated.baseURL)
	}
}

func TestHTTPMCPClientFactory_SweepIdleClients_RemovesOnlyIdleEntries(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	tenantID := uuid.New()

	idleServer := ServerInfo{ID: uuid.New(), Host: "idle.example.com", Port: 443, TLS: true}
	activeServer := ServerInfo{ID: uuid.New(), Host: "active.example.com", Port: 443, TLS: true}

	factory.ClientFor(tenantID, idleServer)
	factory.ClientFor(tenantID, activeServer)

	key := clientCacheKey{tenantID: tenantID, serverID: idleServer.ID}
	factory.cacheMu.Lock()
	factory.cache[key].lastUsedAt.Store(time.Now().Add(-2 * mcpClientIdleTimeout).UnixNano())
	factory.cacheMu.Unlock()

	factory.SweepIdleClients(time.Now())

	factory.cacheMu.RLock()
	_, idleStillPresent := factory.cache[key]
	_, activeStillPresent := factory.cache[clientCacheKey{tenantID: tenantID, serverID: activeServer.ID}]
	factory.cacheMu.RUnlock()

	if idleStillPresent {
		t.Error("expected the idle client to be swept")
	}
	if !activeStillPresent {
		t.Error("expected the recently-used client to remain")
	}
}

func TestHTTPMCPClientFactory_ClientFor_ConcurrentAccessIsSafe(t *testing.T) {
	factory := NewHTTPMCPClientFactory()
	tenantID := uuid.New()
	server := ServerInfo{ID: uuid.New(), Host: "jira.example.com", Port: 443, TLS: true}

	done := make(chan struct{})
	for i := 0; i < 20; i++ {
		go func() {
			factory.ClientFor(tenantID, server)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 20; i++ {
		<-done
	}
}
