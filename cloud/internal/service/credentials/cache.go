// cache.go provides a thread-safe credential cache keyed by credential_provider_id, caching tokens at 80% of their TTL.
package credentials

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type CredentialCache struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*cachedCredential
}

type cachedCredential struct {
	credential *domain.Credential
	expiresAt  time.Time
}

func NewCredentialCache() *CredentialCache {
	return &CredentialCache{
		entries: make(map[uuid.UUID]*cachedCredential),
	}
}

func (c *CredentialCache) Get(id uuid.UUID) (*domain.Credential, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[id]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.credential, true
}

func (c *CredentialCache) Set(id uuid.UUID, cred *domain.Credential, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[id] = &cachedCredential{
		credential: cred,
		expiresAt:  time.Now().Add(ttl),
	}
}

func CacheTTL(providerType string, expiresAt int64) time.Duration {
	switch providerType {
	case "api_key", "username_password":
		return 0
	case "vault_client_token":
		return 5 * time.Minute
	}

	if expiresAt == 0 {
		return 0
	}

	remaining := time.Until(time.Unix(expiresAt, 0))
	if remaining <= 0 {
		return 0
	}

	switch providerType {
	case "aws_sts_federation":
		return remaining
	default:
		return time.Duration(float64(remaining) * 0.8)
	}
}
