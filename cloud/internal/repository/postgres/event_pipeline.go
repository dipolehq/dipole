/*
event_pipeline.go implements the events.Repository interface for the centralized
event pipeline.

This adapter maps the typed event structs from cloud/internal/events to the
existing database tables (access_authorization_events, workload_events,
audit_logs) without schema changes. The payload JSONB column for security
events is built from typed struct fields at write time — never from
map[string]any at the caller side.
*/
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/scope"
)

type EventPipelineRepo struct {
	pool Pool
}

func NewEventPipelineRepo(pool Pool) *EventPipelineRepo {
	return &EventPipelineRepo{pool: pool}
}

func (r *EventPipelineRepo) InsertSecurityEvent(ctx context.Context, event *events.SecurityEvent) error {
	id := uuid.New()

	source := event.Source
	if source == "" {
		source = "edge_proxy"
	}

	payload := buildSecurityPayload(event)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO access_authorization_events (id, tenant_id, event_type, context_id, severity, client_ip, timestamp, payload, resource_set_id, source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		id, event.TenantID, event.EventType, event.ContextID, event.Severity, event.SourceIP, event.Timestamp, payload, event.ResourceSetID, source)
	if err != nil {
		return fmt.Errorf("insert security event: %w", err)
	}
	return nil
}

func (r *EventPipelineRepo) InsertWorkloadEventBatch(ctx context.Context, batch []events.WorkloadEvent) error {
	if len(batch) == 0 {
		return nil
	}

	const cols = 12
	values := make([]string, 0, len(batch))
	args := make([]interface{}, 0, len(batch)*cols)

	for i, e := range batch {
		base := i * cols
		values = append(values, fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6,
			base+7, base+8, base+9, base+10, base+11, base+12))

		payload := e.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}

		args = append(args,
			uuid.New(),
			e.TenantID,
			e.ConnectionID,
			e.EventType,
			e.PolicyID,
			e.ClientWorkloadID,
			e.ServerWorkloadID,
			e.Outcome,
			e.Severity,
			e.Timestamp,
			payload,
			e.ResourceSetID,
		)
	}

	query := `INSERT INTO workload_events (id, tenant_id, connection_id, event_type, policy_id, client_workload_id, server_workload_id, outcome, severity, timestamp, payload, resource_set_id)
		 VALUES ` + strings.Join(values, ",")

	_, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert workload event batch: %w", err)
	}
	return nil
}

func (r *EventPipelineRepo) InsertAuditEvent(ctx context.Context, event *events.AuditEvent) error {
	id := uuid.New()

	severity := "Info"

	_, err := r.pool.Exec(ctx,
		`INSERT INTO audit_logs (id, tenant_id, actor_email, category, activity, target_type, target_id, target_name, severity, client_ip, timestamp)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		id, event.TenantID, event.ActorEmail, event.Category, event.Activity, event.TargetType, event.TargetID, event.TargetName, severity, event.ActorIP, event.Timestamp)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (r *EventPipelineRepo) QuerySecurityEvents(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) ([]events.SecurityEvent, int, error) {
	where, args := buildPipelineAuthFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM access_authorization_events WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count security events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, event_type, context_id, severity, client_ip, timestamp, payload, resource_set_id, source
		 FROM access_authorization_events WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query security events: %w", err)
	}
	defer rows.Close()

	var results []events.SecurityEvent
	for rows.Next() {
		var (
			id          uuid.UUID
			tid         uuid.UUID
			eventType   string
			contextID   uuid.UUID
			severity    string
			clientIP    string
			timestamp   time.Time
			payload     json.RawMessage
			resourceSet *uuid.UUID
			source      string
		)
		if err := rows.Scan(&id, &tid, &eventType, &contextID, &severity, &clientIP, &timestamp, &payload, &resourceSet, &source); err != nil {
			return nil, 0, fmt.Errorf("scan security event: %w", err)
		}
		results = append(results, events.SecurityEvent{
			TenantID:      tid,
			ContextID:     contextID,
			Timestamp:     timestamp,
			SourceIP:      clientIP,
			Source:        source,
			EventType:     eventType,
			Severity:      severity,
			ResourceSetID: resourceSet,
		})
	}
	return results, total, nil
}

func (r *EventPipelineRepo) QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) ([]events.WorkloadEvent, int, error) {
	where, args := buildPipelineWorkloadFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workload_events WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count workload events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, connection_id, event_type, policy_id, client_workload_id, server_workload_id, outcome, severity, timestamp, payload, resource_set_id
		 FROM workload_events WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query workload events: %w", err)
	}
	defer rows.Close()

	var results []events.WorkloadEvent
	for rows.Next() {
		var (
			id          uuid.UUID
			tid         uuid.UUID
			connID      uuid.UUID
			eventType   string
			policyID    *uuid.UUID
			cwID        *uuid.UUID
			swID        *uuid.UUID
			outcome     string
			severity    string
			timestamp   time.Time
			payload     json.RawMessage
			resourceSet *uuid.UUID
		)
		_ = id
		if err := rows.Scan(&id, &tid, &connID, &eventType, &policyID, &cwID, &swID, &outcome, &severity, &timestamp, &payload, &resourceSet); err != nil {
			return nil, 0, fmt.Errorf("scan workload event: %w", err)
		}
		results = append(results, events.WorkloadEvent{
			TenantID:         tid,
			ConnectionID:     connID,
			Timestamp:        timestamp,
			EventType:        eventType,
			Severity:         severity,
			Outcome:          outcome,
			PolicyID:         policyID,
			ClientWorkloadID: cwID,
			ServerWorkloadID: swID,
			ResourceSetID:    resourceSet,
			Payload:          payload,
		})
	}
	return results, total, nil
}

func (r *EventPipelineRepo) QueryAuditEvents(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) ([]events.AuditEvent, int, error) {
	where, args := buildPipelineAuditFilters(ctx, tenantID, opts)

	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_logs WHERE `+where, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}

	offset := (opts.Page - 1) * opts.PerPage
	args = append(args, opts.PerPage, offset)
	limitParam := fmt.Sprintf("$%d", len(args)-1)
	offsetParam := fmt.Sprintf("$%d", len(args))

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, actor_email, category, activity, target_type, target_id, target_name, severity, client_ip, timestamp
		 FROM audit_logs WHERE `+where+` ORDER BY timestamp DESC, id DESC LIMIT `+limitParam+` OFFSET `+offsetParam,
		args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var results []events.AuditEvent
	for rows.Next() {
		var (
			id        uuid.UUID
			tid       uuid.UUID
			email     string
			category  string
			activity  string
			tType     string
			tID       *uuid.UUID
			tName     string
			severity  string
			clientIP  string
			timestamp time.Time
		)
		_ = id
		if err := rows.Scan(&id, &tid, &email, &category, &activity, &tType, &tID, &tName, &severity, &clientIP, &timestamp); err != nil {
			return nil, 0, fmt.Errorf("scan audit event: %w", err)
		}
		results = append(results, events.AuditEvent{
			TenantID:   tid,
			Timestamp:  timestamp,
			ActorEmail: email,
			ActorIP:    clientIP,
			Category:   category,
			Activity:   activity,
			TargetType: tType,
			TargetID:   tID,
			TargetName: tName,
		})
	}
	return results, total, nil
}

func buildSecurityPayload(event *events.SecurityEvent) json.RawMessage {
	switch event.EventType {
	case "access.request":
		return buildRequestPayload(event)
	case "access.authorization":
		return buildAuthorizationPayload(event)
	case "access.credential":
		return buildCredentialPayload(event)
	case "access.discovery":
		return buildDiscoveryPayload(event)
	default:
		return buildAuthorizationPayload(event)
	}
}

func buildRequestPayload(event *events.SecurityEvent) json.RawMessage {
	payload := map[string]any{
		"clientRequest": map[string]any{
			"version": "1.0.0",
			"network": map[string]any{
				"sourceIP":          event.SourceIP,
				"sourcePort":        0,
				"transportProtocol": "TCP",
				"proxyPort":         8080,
				"targetHost":        event.ServerHost,
				"targetPort":        event.ServerPort,
			},
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildAuthorizationPayload(event *events.SecurityEvent) json.RawMessage {
	outcome := map[string]any{"result": "Error"}
	if event.Allowed {
		outcome["result"] = "Authorized"
	} else if event.DenyReason != "" {
		outcome["result"] = "Unauthorized"
		outcome["reason"] = event.DenyReason
	}

	cwBlock := map[string]any{"result": "NotFound"}
	if event.ClientWorkload != "" {
		cwBlock = map[string]any{"name": event.ClientWorkload, "result": "Identified"}
		if event.ClientWorkloadID != nil {
			cwBlock["id"] = event.ClientWorkloadID.String()
		}
	}

	swBlock := map[string]any{"result": "NotFound"}
	if event.ServerWorkload != "" {
		swBlock = map[string]any{"name": event.ServerWorkload, "result": "Identified"}
		if event.ServerWorkloadID != nil {
			swBlock["id"] = event.ServerWorkloadID.String()
		}
	}

	apBlock := map[string]any{"result": "NotFound"}
	if event.PolicyName != "" {
		apBlock = map[string]any{"name": event.PolicyName, "result": "Identified"}
		if event.PolicyID != nil {
			apBlock["id"] = event.PolicyID.String()
		}
	}

	tpBlocks := []map[string]any{}
	if len(event.TrustResults) > 0 {
		for _, tr := range event.TrustResults {
			tp := map[string]any{
				"name": tr.ProviderName,
			}
			if tr.ProviderID != (uuid.UUID{}) {
				tp["id"] = tr.ProviderID.String()
			}
			if tr.Attested {
				tp["result"] = "Attested"
			} else {
				tp["result"] = "Unauthorized"
				tp["reason"] = tr.Reason
				if tr.Attribute != "" {
					tp["attribute"] = tr.Attribute
					tp["expectedValue"] = tr.ExpectedValue
					tp["actualValue"] = tr.ActualValue
				}
			}
			tpBlocks = append(tpBlocks, tp)
		}
	} else if event.TrustProvider != "" {
		tp := map[string]any{"name": event.TrustProvider}
		if event.TrustAttested {
			tp["result"] = "Attested"
		} else {
			tp["result"] = "Unauthorized"
		}
		tpBlocks = append(tpBlocks, tp)
	}

	cpBlock := map[string]any{}
	if event.CredentialName != "" {
		cpBlock["name"] = event.CredentialName
		cpBlock["type"] = event.CredentialType
		if event.CredentialProviderID != nil {
			cpBlock["id"] = event.CredentialProviderID.String()
		}
		if event.Allowed {
			cpBlock["result"] = "Retrieved"
			if event.CredentialMaxAge > 0 {
				cpBlock["maxAge"] = event.CredentialMaxAge
			}
		} else {
			cpBlock["result"] = "Identified"
		}
	}

	payload := map[string]any{
		"outcome":            outcome,
		"clientWorkload":     cwBlock,
		"serverWorkload":     swBlock,
		"accessPolicy":       apBlock,
		"trustProviders":     tpBlocks,
		"accessConditions":   []map[string]any{},
		"credentialProvider": cpBlock,
	}

	if event.ToolName != "" {
		payload["toolName"] = event.ToolName
	}
	if event.UserID != "" {
		payload["userId"] = event.UserID
	}

	data, _ := json.Marshal(payload)
	return data
}

func buildCredentialPayload(event *events.SecurityEvent) json.RawMessage {
	outcome := map[string]any{"result": "Unauthorized"}
	if event.Allowed {
		outcome["result"] = "Authorized"
	}

	cwBlock := map[string]any{"result": "NotFound"}
	if event.ClientWorkload != "" {
		cwBlock = map[string]any{"name": event.ClientWorkload, "result": "Identified"}
		if event.ClientWorkloadID != nil {
			cwBlock["id"] = event.ClientWorkloadID.String()
		}
	}

	swBlock := map[string]any{"result": "NotFound"}
	if event.ServerWorkload != "" {
		swBlock = map[string]any{"name": event.ServerWorkload, "result": "Identified"}
		if event.ServerWorkloadID != nil {
			swBlock["id"] = event.ServerWorkloadID.String()
		}
	}

	apBlock := map[string]any{"result": "NotFound"}
	if event.PolicyName != "" {
		apBlock = map[string]any{"name": event.PolicyName, "result": "Identified"}
		if event.PolicyID != nil {
			apBlock["id"] = event.PolicyID.String()
		}
	}

	tpBlocks := []map[string]any{}
	if len(event.TrustResults) > 0 {
		for _, tr := range event.TrustResults {
			tp := map[string]any{"name": tr.ProviderName}
			if tr.ProviderID != (uuid.UUID{}) {
				tp["id"] = tr.ProviderID.String()
			}
			if tr.Attested {
				tp["result"] = "Attested"
			} else {
				tp["result"] = "Unauthorized"
				tp["reason"] = tr.Reason
			}
			tpBlocks = append(tpBlocks, tp)
		}
	}

	cpBlock := map[string]any{}
	if event.CredentialName != "" {
		cpBlock["type"] = event.CredentialType
		cpBlock["name"] = event.CredentialName
		if event.CredentialProviderID != nil {
			cpBlock["id"] = event.CredentialProviderID.String()
		}
		if event.Allowed {
			cpBlock["result"] = "Retrieved"
			if event.CredentialMaxAge > 0 {
				cpBlock["maxAge"] = event.CredentialMaxAge
			}
		} else {
			cpBlock["result"] = "Failed"
			if event.DenyReason != "" {
				cpBlock["reason"] = event.DenyReason
			}
		}
	}

	payload := map[string]any{
		"outcome":            outcome,
		"clientWorkload":     cwBlock,
		"serverWorkload":     swBlock,
		"accessPolicy":       apBlock,
		"trustProviders":     tpBlocks,
		"accessConditions":   []map[string]any{},
		"credentialProvider": cpBlock,
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildDiscoveryPayload(event *events.SecurityEvent) json.RawMessage {
	clientWorkloads := []map[string]any{}
	for _, cw := range event.CandidateClientWorkloads {
		clientWorkloads = append(clientWorkloads, map[string]any{
			"id":   cw.ID.String(),
			"name": cw.Name,
		})
	}

	serverWorkloads := []map[string]any{}
	for _, sw := range event.CandidateServerWorkloads {
		serverWorkloads = append(serverWorkloads, map[string]any{
			"id":   sw.ID.String(),
			"name": sw.Name,
		})
	}

	payload := map[string]any{
		"clientWorkloads": clientWorkloads,
		"serverWorkloads": serverWorkloads,
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildPipelineAuthFilters(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(event_type ILIKE '%%' || $%d || '%%' OR context_id::text ILIKE '%%' || $%d || '%%' OR client_ip ILIKE '%%' || $%d || '%%')", n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("event_type = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.Source != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("source = $%d", n))
		args = append(args, opts.Source)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}

	if s := scope.FromContext(ctx); s != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("resource_set_id = $%d", n))
		args = append(args, s.ResourceSetID)
	}

	return strings.Join(clauses, " AND "), args
}

func buildPipelineWorkloadFilters(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(event_type ILIKE '%%' || $%d || '%%' OR connection_id::text ILIKE '%%' || $%d || '%%' OR outcome ILIKE '%%' || $%d || '%%')", n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("event_type = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}
	if len(opts.ClientWorkloadIDs) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("client_workload_id::text = ANY($%d)", n))
		args = append(args, opts.ClientWorkloadIDs)
	}
	if len(opts.ServerWorkloadIDs) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("server_workload_id::text = ANY($%d)", n))
		args = append(args, opts.ServerWorkloadIDs)
	}

	if s := scope.FromContext(ctx); s != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("resource_set_id = $%d", n))
		args = append(args, s.ResourceSetID)
	}

	return strings.Join(clauses, " AND "), args
}

func buildPipelineAuditFilters(ctx context.Context, tenantID uuid.UUID, opts events.QueryOptions) (string, []interface{}) {
	args := []interface{}{tenantID}
	clauses := []string{"tenant_id = $1"}
	n := 1

	if opts.Search != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("(actor_email ILIKE '%%' || $%d || '%%' OR category ILIKE '%%' || $%d || '%%' OR activity ILIKE '%%' || $%d || '%%' OR target_name ILIKE '%%' || $%d || '%%')", n, n, n, n))
		args = append(args, opts.Search)
	}
	if len(opts.Severities) > 0 {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = ANY($%d)", n))
		args = append(args, opts.Severities)
	} else if opts.Severity != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("severity = $%d", n))
		args = append(args, opts.Severity)
	}
	if opts.EventType != "" {
		n++
		clauses = append(clauses, fmt.Sprintf("category = $%d", n))
		args = append(args, opts.EventType)
	}
	if opts.StartDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp >= $%d", n))
		args = append(args, *opts.StartDate)
	}
	if opts.EndDate != nil {
		n++
		clauses = append(clauses, fmt.Sprintf("timestamp <= $%d", n))
		args = append(args, *opts.EndDate)
	}

	return strings.Join(clauses, " AND "), args
}
