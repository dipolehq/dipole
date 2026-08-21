// Package domain defines core business entities and value objects.
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
