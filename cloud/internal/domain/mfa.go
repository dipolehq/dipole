// Package domain defines core business entities and value objects.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserMFA struct {
	ID                   uuid.UUID
	TenantID             uuid.UUID
	UserID               uuid.UUID
	TOTPSecretEncrypted  []byte
	IsEnabled            bool
	BackupCodesEncrypted []byte
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
