// Package domain defines core business entities and value objects.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type RegistrationToken struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	TokenHash    string
	ControllerID uuid.UUID
	Consumed     bool
	ExpiresAt    time.Time
	CreatedAt    time.Time
}
