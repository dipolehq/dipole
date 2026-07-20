package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Tenant struct {
	ID           uuid.UUID
	Name         string
	Subdomain    string
	APITokenHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Identifier struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type ClientWorkload struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Name          string
	Description   string
	IsActive      bool
	Identifiers   []Identifier
	Tags          []Tag
	ResourceSetID *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time

	// PolicyCount is a derived field populated only by list queries; it is the
	// number of access policies referencing this workload.
	PolicyCount int
}

type ServerWorkload struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	Name              string
	Description       string
	IsActive          bool
	Host              string
	Port              int
	AppProtocol       string
	TransportProtocol string
	TLS               bool
	TLSVerification   string
	RequestedPort     int
	RequestedTLS      bool
	AuthMethod        string
	AuthScheme        string
	AuthConfig        map[string]string
	HTTPHeaders       map[string]string
	URLPath           string
	Tags              []Tag
	ResourceSetID     *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time

	// PolicyCount is a derived field populated only by list queries; it is the
	// number of access policies referencing this workload.
	PolicyCount int
}

type MatchRule struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

type TrustProvider struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Description     string
	IsActive        bool
	Type            string
	Config          json.RawMessage
	MatchRules      []MatchRule
	EdgeSDKClientID string
	EdgeSDKAudience string
	Tags            []Tag
	ResourceSetID   *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PolicyCount     int
}

type CredentialProvider struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Description     string
	IsActive        bool
	Type            string
	ConfigEncrypted []byte
	Tags            []Tag
	IntegrationID   *uuid.UUID
	ResourceSetID   *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PolicyCount     int
}

type AccessCondition struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	Name          string
	Description   string
	IsActive      bool
	Type          string
	Config        json.RawMessage
	Tags          []Tag
	ResourceSetID *uuid.UUID
	CreatedAt     time.Time
	UpdatedAt     time.Time
	PolicyCount   int
}

type PolicyCredentialProvider struct {
	CredentialProviderID uuid.UUID
	MappingType          string
	MappingValue         string
}

type AccessPolicy struct {
	ID                    uuid.UUID
	TenantID              uuid.UUID
	Name                  string
	Description           string
	IsActive              bool
	ClientWorkloadID      uuid.UUID
	ServerWorkloadID      uuid.UUID
	TrustProviderIDs      []uuid.UUID
	CredentialProviderIDs []PolicyCredentialProvider
	AccessConditionIDs    []uuid.UUID
	Tags                  []Tag
	ResourceSetID         *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type AccessPolicyWithRelations struct {
	Policy              AccessPolicy
	ClientWorkload      ClientWorkload
	ServerWorkload      ServerWorkload
	TrustProviders      []TrustProvider
	CredentialProviders []CredentialProvider
	AccessConditions    []AccessCondition
}

type CredentialProviderIntegration struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Description     string
	Type            string
	ConfigEncrypted []byte
	IsActive        bool
	Tags            []Tag
	ResourceSetID   *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type AgentController struct {
	ID                  uuid.UUID
	TenantID            uuid.UUID
	Name                string
	Description         string
	IsActive            bool
	Status              string
	DeviceCode          string
	DeviceCodeExpiresAt *time.Time
	TrustProviderID     *uuid.UUID
	TLSHostname         string
	Tags                []Tag
	LastHealthAt        *time.Time
	ResourceSetID       *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type AgentProxy struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	AgentControllerID *uuid.UUID
	Status            string
	LastHealthAt      *time.Time
	ResourceSetID     *uuid.UUID
	RegisteredAt      *time.Time
}

type ResourceSet struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
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

type User struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Email     string
	Name      string
	FirstName string
	LastName  string
	Phone     string
	IsActive  bool
	RoleID    *uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

type IdentityProvider struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	IsActive    bool
	Type        string // "saml_2.0" or "oidc_1.0"
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

type ListOptions struct {
	Page    int
	PerPage int
	OrderBy string
	Search  string
}
