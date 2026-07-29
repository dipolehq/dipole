package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AuthEvent struct {
	ID            uuid.UUID       `json:"id"`
	TenantID      uuid.UUID       `json:"tenantId"`
	EventType     string          `json:"eventType"`
	ContextID     uuid.UUID       `json:"contextId"`
	Severity      string          `json:"severity"`
	ClientIP      string          `json:"clientIp"`
	Timestamp     time.Time       `json:"timestamp"`
	Payload       json.RawMessage `json:"payload"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId"`
}

type WorkloadEvent struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         uuid.UUID       `json:"tenantId"`
	ConnectionID     uuid.UUID       `json:"connectionId"`
	EventType        string          `json:"eventType"`
	PolicyID         *uuid.UUID      `json:"policyId"`
	ClientWorkloadID *uuid.UUID      `json:"clientWorkloadId"`
	ServerWorkloadID *uuid.UUID      `json:"serverWorkloadId"`
	Outcome          string          `json:"outcome"`
	Severity         string          `json:"severity"`
	Timestamp        time.Time       `json:"timestamp"`
	Payload          json.RawMessage `json:"payload"`
	ResourceSetID    *uuid.UUID      `json:"resourceSetId"`
}

type AuditLog struct {
	ID         uuid.UUID  `json:"id"`
	TenantID   uuid.UUID  `json:"tenantId"`
	ActorID    *uuid.UUID `json:"actorId"`
	ActorEmail string     `json:"actorEmail"`
	Category   string     `json:"category"`
	Activity   string     `json:"activity"`
	TargetType string     `json:"targetType"`
	TargetID   *uuid.UUID `json:"targetId"`
	TargetName string     `json:"targetName"`
	Result     string     `json:"result"`
	Severity   string     `json:"severity"`
	ClientIP   string     `json:"clientIp"`
	Browser    string     `json:"browser"`
	OS         string     `json:"os"`
	Timestamp  time.Time  `json:"timestamp"`
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

type DashboardEventData struct {
	TotalLast6h          int
	AuthorizedLast6h     int
	DeniedLast6h         int
	TimeSeries           []DashboardTimePoint
	AccessDecisionSeries []DashboardDecisionPoint
	TopClients           []DashboardWorkloadCount
	TopServers           []DashboardWorkloadCount
	CredentialUsage      []DashboardTypeCount
	ProtocolUsage        []DashboardTypeCount
	PolicyFailures       []DashboardPolicyFailure
}

type DashboardTimePoint struct {
	Timestamp time.Time
	Info      int
	Warning   int
	Error     int
}

type DashboardDecisionPoint struct {
	Timestamp  time.Time
	Authorized int
	Denied     int
}

type DashboardWorkloadCount struct {
	ID    string
	Name  string
	Count int
}

type DashboardTypeCount struct {
	Type  string
	Count int
}

type DashboardPolicyFailure struct {
	PolicyID   string
	PolicyName string
	Count      int
}
