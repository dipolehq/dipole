// Package events defines typed event structs for Security, Workload, and Audit pipelines.
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SecurityEvent struct {
	TenantID      uuid.UUID
	ContextID     uuid.UUID
	Timestamp     time.Time
	SourceIP      string
	Source        string
	EventType     string
	Severity      string
	ResourceSetID *uuid.UUID
	Detail        SecurityEventDetail
}

type SecurityEventDetail interface {
	securityEventDetail()
}

type RequestDetail struct {
	ServerHost string
	ServerPort int
}

func (RequestDetail) securityEventDetail() {}

type AuthorizationDetail struct {
	Allowed              bool
	DenyReason           string
	ClientWorkload       string
	ClientWorkloadID     *uuid.UUID
	ServerWorkload       string
	ServerWorkloadID     *uuid.UUID
	PolicyID             *uuid.UUID
	PolicyName           string
	TrustProvider        string
	TrustAttested        bool
	TrustResults         []TrustResult
	CredentialType       string
	CredentialName       string
	CredentialProviderID *uuid.UUID
	CredentialMaxAge     int
	ToolName             string
	UserID               string
}

func (AuthorizationDetail) securityEventDetail() {}

type CredentialDetail struct {
	Allowed              bool
	DenyReason           string
	ClientWorkload       string
	ClientWorkloadID     *uuid.UUID
	ServerWorkload       string
	ServerWorkloadID     *uuid.UUID
	PolicyID             *uuid.UUID
	PolicyName           string
	TrustResults         []TrustResult
	CredentialType       string
	CredentialName       string
	CredentialProviderID *uuid.UUID
	CredentialMaxAge     int
}

func (CredentialDetail) securityEventDetail() {}

type DiscoveryDetail struct {
	CandidateClientWorkloads []WorkloadRef
	CandidateServerWorkloads []WorkloadRef
}

func (DiscoveryDetail) securityEventDetail() {}

type TrustResult struct {
	ProviderID    uuid.UUID
	ProviderName  string
	Attested      bool
	Reason        string
	Attribute     string
	ExpectedValue string
	ActualValue   string
}

type WorkloadRef struct {
	ID   uuid.UUID
	Name string
}

type WorkloadEvent struct {
	EventID          uuid.UUID
	TenantID         uuid.UUID
	ConnectionID     uuid.UUID
	Timestamp        time.Time
	EventType        string
	Severity         string
	ClientWorkload   string
	ServerWorkload   string
	ServerHost       string
	ServerPort       int
	Outcome          string
	HTTPMethod       string
	HTTPPath         string
	HTTPStatus       int
	UserID           string
	Latency          time.Duration
	PolicyID         *uuid.UUID
	ClientWorkloadID *uuid.UUID
	ServerWorkloadID *uuid.UUID
	ResourceSetID    *uuid.UUID
	Payload          json.RawMessage
}

type AuditEvent struct {
	TenantID      uuid.UUID
	Timestamp     time.Time
	ActorID       *uuid.UUID
	ActorEmail    string
	ActorIP       string
	Browser       string
	OS            string
	Category      string
	Activity      string
	Severity      string
	TargetType    string
	TargetID      *uuid.UUID
	TargetName    string
	ResourceSetID *uuid.UUID
}
