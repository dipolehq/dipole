/*
Package events defines the centralized event model for the Dipole access
control platform.

model.go contains all typed event structs used across the four event pipelines
(Security, Workload, Tracing, Audit). These structs are the canonical
representations — every producer constructs them, every consumer reads them.
No map[string]any or json.RawMessage is used for core fields.
*/
package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SecurityEvent is the envelope shared by all four access.* event types
// (access.request, access.authorization, access.credential, access.discovery).
// Only fields that apply to every event type regardless of what happened
// live here; everything specific to one event type lives on Detail, in the
// SecurityEventDetail implementation matching EventType. This means a
// producer can only set fields that apply to the event it is building —
// there is no flat struct with unrelated fields sitting at zero-value.
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

// SecurityEventDetail is implemented by exactly four types — RequestDetail,
// AuthorizationDetail, CredentialDetail, DiscoveryDetail — one per access.*
// EventType. The marker method is unexported so this set stays closed to
// this package: producers construct one of the four existing types rather
// than inventing a fifth event shape.
type SecurityEventDetail interface {
	securityEventDetail()
}

// RequestDetail is the Detail for an "access.request" event: the raw
// inbound connection info before any policy/identity decision is made.
type RequestDetail struct {
	ServerHost string
	ServerPort int
}

func (RequestDetail) securityEventDetail() {}

// AuthorizationDetail is the Detail for an "access.authorization" event:
// the full access decision — which client/server/policy/trust
// provider(s)/credential provider were involved and the outcome.
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

// CredentialDetail is the Detail for an "access.credential" event: whether
// a credential was actually retrieved for an already-authorized request.
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

// DiscoveryDetail is the Detail for an "access.discovery" event: which
// client/server workloads were candidates for this request, independent of
// whether any of them ultimately matched.
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

type TraceEntry struct {
	ID             string              `json:"id"`
	TenantID       uuid.UUID           `json:"-"`
	Timestamp      time.Time           `json:"timestamp"`
	ClientIP       string              `json:"clientIP"`
	AccessPolicy   TracePolicyStatus   `json:"accessPolicy"`
	RedirectURI    string              `json:"redirectUri"`
	ClientWorkload TraceWorkloadStatus `json:"clientWorkload"`
	ResourceServer string              `json:"resourceServer"`
	ServerWorkload TraceWorkloadStatus `json:"serverWorkload"`
	Severity       string              `json:"severity"`
}

type TracePolicyStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type TraceWorkloadStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type AuditEvent struct {
	TenantID      uuid.UUID
	Timestamp     time.Time
	ActorEmail    string
	ActorIP       string
	Browser       string
	OS            string
	Category      string
	Activity      string
	TargetType    string
	TargetID      *uuid.UUID
	TargetName    string
	ResourceSetID *uuid.UUID
}
