/*
password_reset.go defines the PasswordResetToken domain type for the
"forgot password" flow. Tokens are SHA-256 hashed on creation and stored
with a 1-hour expiry. Once used, the token is marked consumed and cannot
be replayed.
*/
package domain

import (
	"time"

	"github.com/google/uuid"
)

type PasswordResetToken struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}
