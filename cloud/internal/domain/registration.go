/*
registration.go defines the RegistrationToken domain type for single-use
proxy registration tokens. Tokens are hashed (SHA-256) on issuance and
stored with an expiry. Consumed tokens cannot be replayed.
*/
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
