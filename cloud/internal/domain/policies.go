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
