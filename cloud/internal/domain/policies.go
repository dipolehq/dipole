// Package domain defines core business entities and value objects.
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MatchRule struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

type TrustProvider struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string
	Description string
	IsActive    bool
	Type        string
	// Config holds non-secret attestation settings (OIDC endpoint, JWKS,
	// public key, attestation method, etc). It must never carry the
	// symmetric_key attestation secret — that lives encrypted in
	// SymmetricKeyEncrypted instead.
	Config json.RawMessage
	// SymmetricKeyEncrypted holds the AES-256-GCM encrypted HMAC shared
	// secret for the "symmetric_key" OIDC attestation method, or nil if
	// this provider doesn't use that method. Decrypted only at validation
	// time (service/trust) or when a caller explicitly re-encrypts an
	// updated value (service/resources) — never returned to API clients.
	SymmetricKeyEncrypted []byte
	MatchRules            []MatchRule
	EdgeSDKClientID       string
	EdgeSDKAudience       string
	Tags                  []Tag
	ResourceSetID         *uuid.UUID
	CreatedAt             time.Time
	UpdatedAt             time.Time
	PolicyCount           int
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
