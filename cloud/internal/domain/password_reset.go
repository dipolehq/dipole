// Package domain defines core business entities and value objects.
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
