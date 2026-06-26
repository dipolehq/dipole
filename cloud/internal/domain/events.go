package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuthEvent struct {
	ID            uuid.UUID
	TenantID      uuid.UUID
	EventType     string
	ContextID     uuid.UUID
	Severity      string
	ClientIP      string
	Timestamp     time.Time
	Payload       json.RawMessage
	ResourceSetID *uuid.UUID
}

type WorkloadEvent struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	ConnectionID     uuid.UUID
	EventType        string
	PolicyID         *uuid.UUID
	ClientWorkloadID *uuid.UUID
	ServerWorkloadID *uuid.UUID
	Outcome          string
	Severity         string
	Timestamp        time.Time
	Payload          json.RawMessage
	ResourceSetID    *uuid.UUID
}

type AuditLog struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	ActorID    *uuid.UUID
	ActorEmail string
	Category   string
	Activity   string
	TargetType string
	TargetID   *uuid.UUID
	TargetName string
	Result     string
	Severity   string
	ClientIP   string
	Browser    string
	OS         string
	Timestamp  time.Time
}

type EventQueryOptions struct {
	Page              int
	PerPage           int
	OrderBy           string
	Search            string
	StartDate         *time.Time
	EndDate           *time.Time
	Severity          string
	EventType         string
	ClientWorkloadIDs []string
	ServerWorkloadIDs []string
	UserID            string
}
