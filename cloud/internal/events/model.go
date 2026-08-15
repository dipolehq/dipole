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

type SecurityEvent struct {
	TenantID       uuid.UUID
	ContextID      uuid.UUID
	Timestamp      time.Time
	SourceIP       string
	Source         string
	EventType      string
	Severity       string
	Allowed        bool
	DenyReason     string
	PolicyID       *uuid.UUID
	PolicyName     string
	UserID         string
	ClientWorkload string
	ServerWorkload string
	ServerHost     string
	ServerPort     int
	TrustProvider  string
	TrustAttested  bool
	CredentialType string
	CredentialName string
	ToolName       string
	ResourceSetID  *uuid.UUID
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
	ID             string
	TenantID       uuid.UUID
	Timestamp      time.Time
	ClientIP       string
	AccessPolicy   TracePolicyStatus
	RedirectURI    string
	ClientWorkload TraceWorkloadStatus
	ResourceServer string
	ServerWorkload TraceWorkloadStatus
	Severity       string
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
