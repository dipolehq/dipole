package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID           uuid.UUID
	Name         string
	Subdomain    string
	Plan         string
	APITokenHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type TenantSummary struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Subdomain string    `json:"subdomain"`
}

type TenantProfile struct {
	TenantID  uuid.UUID
	FirstName string
	LastName  string
	Email     string
	Phone     string
	UpdatedAt time.Time
}

type User struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Email        string
	Name         string
	FirstName    string
	LastName     string
	Phone        string
	IsActive     bool
	RoleID       *uuid.UUID
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Role struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	IsActive    bool
	Permissions json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ResourceSet struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ResourceSetCounts holds the number of each resource type within a resource set.
type ResourceSetCounts struct {
	AccessPolicies      int
	ClientWorkloads     int
	TrustProviders      int
	AccessConditions    int
	CredentialProviders int
	ServerWorkloads     int
}

type IdentityProvider struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	IsActive    bool
	Type        string
	Config      json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type SignOnPolicy struct {
	TenantID   uuid.UUID
	RequireSSO bool
	RequireMFA bool
	UpdatedAt  time.Time
}

type GlobalPolicyCompliance struct {
	TenantID                   uuid.UUID
	APTrustProvider            string
	APAccessCondition          string
	ACTrustProvider            string
	ACTLSHostname              string
	AccessTokenLifetimeSeconds int
	UpdatedAt                  time.Time
}

type AgentController struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	Name                  string
	Description           string
	IsActive              bool
	Status                string
	DeviceCode            string
	DeviceCodeExpiresAt   *time.Time
	TrustProviderID       *uuid.UUID
	TLSHostname           string
	SharedSecretEncrypted []byte
	Tags                  []Tag
	LastHealthAt          *time.Time
	ResourceSetID         *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AgentProxy struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	AgentControllerID     *uuid.UUID
	Status                string
	LastHealthAt          *time.Time
	ResourceSetID         *uuid.UUID
	RegisteredAt          *time.Time
	SessionTokenExpiresAt *time.Time
}

type LogStream struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Description     string
	IsActive        bool
	EventType       string
	DestinationType string
	Config          json.RawMessage
	LastExportedAt  *time.Time
	LastExportError *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ListOptions struct {
	Page    int
	PerPage int
	OrderBy string
	Search  string
}
