/*
api_token.go defines the APIToken domain type for scoped API tokens.
Each token belongs to a tenant and optionally scopes access to a specific
Resource Set. Tokens with nil ResourceSetID have full tenant-wide access.
*/
package domain

import (
	"time"

	"github.com/google/uuid"
)

type APIToken struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Name          string
	TokenHash     string
	ResourceSetID *uuid.UUID
	CreatedAt     time.Time
}
