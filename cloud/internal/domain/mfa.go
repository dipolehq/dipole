/*
mfa.go defines the UserMFA domain type representing a user's multi-factor
authentication configuration (TOTP secret + backup codes). Secrets and codes
are stored encrypted (AES-256-GCM) at rest.
*/
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
