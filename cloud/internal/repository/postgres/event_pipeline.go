// event_pipeline.go implements the write side of events.Repository (InsertSecurityEvent/InsertWorkloadEventBatch/InsertAuditEvent), mapping typed event structs from cloud/internal/events to the existing access_authorization_events/workload_events/audit_logs tables.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/events"
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

func buildSecurityPayload(event *events.SecurityEvent) json.RawMessage {
	switch event.EventType {
	case "access.request":
		detail, _ := event.Detail.(events.RequestDetail)
		return buildRequestPayload(event.SourceIP, detail)
	case "access.authorization":
		detail, _ := event.Detail.(events.AuthorizationDetail)
		return buildAuthorizationPayload(detail)
	case "access.credential":
		detail, _ := event.Detail.(events.CredentialDetail)
		return buildCredentialPayload(detail)
	case "access.discovery":
		detail, _ := event.Detail.(events.DiscoveryDetail)
		return buildDiscoveryPayload(detail)
	default:
		detail, _ := event.Detail.(events.AuthorizationDetail)
		return buildAuthorizationPayload(detail)
	}
}

func buildRequestPayload(sourceIP string, detail events.RequestDetail) json.RawMessage {
	payload := map[string]any{
		"clientRequest": map[string]any{
			"version": "1.0.0",
			"network": map[string]any{
				"sourceIP":          sourceIP,
				"sourcePort":        0,
				"transportProtocol": "TCP",
				"proxyPort":         8080,
				"targetHost":        detail.ServerHost,
				"targetPort":        detail.ServerPort,
			},
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildAuthorizationPayload(detail events.AuthorizationDetail) json.RawMessage {
	outcome := map[string]any{"result": "Error"}
	if detail.Allowed {
		outcome["result"] = "Authorized"
	} else if detail.DenyReason != "" {
		outcome["result"] = "Unauthorized"
		outcome["reason"] = detail.DenyReason
	}

	cwBlock := map[string]any{"result": "NotFound"}
	if detail.ClientWorkload != "" {
		cwBlock = map[string]any{"name": detail.ClientWorkload, "result": "Identified"}
		if detail.ClientWorkloadID != nil {
			cwBlock["id"] = detail.ClientWorkloadID.String()
		}
	}

	swBlock := map[string]any{"result": "NotFound"}
	if detail.ServerWorkload != "" {
		swBlock = map[string]any{"name": detail.ServerWorkload, "result": "Identified"}
		if detail.ServerWorkloadID != nil {
			swBlock["id"] = detail.ServerWorkloadID.String()
		}
	}

	apBlock := map[string]any{"result": "NotFound"}
	if detail.PolicyName != "" {
		apBlock = map[string]any{"name": detail.PolicyName, "result": "Identified"}
		if detail.PolicyID != nil {
			apBlock["id"] = detail.PolicyID.String()
		}
	}

	tpBlocks := []map[string]any{}
	if len(detail.TrustResults) > 0 {
		for _, tr := range detail.TrustResults {
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
	} else if detail.TrustProvider != "" {
		tp := map[string]any{"name": detail.TrustProvider}
		if detail.TrustAttested {
			tp["result"] = "Attested"
		} else {
			tp["result"] = "Unauthorized"
		}
		tpBlocks = append(tpBlocks, tp)
	}

	cpBlock := map[string]any{}
	if detail.CredentialName != "" {
		cpBlock["name"] = detail.CredentialName
		cpBlock["type"] = detail.CredentialType
		if detail.CredentialProviderID != nil {
			cpBlock["id"] = detail.CredentialProviderID.String()
		}
		if detail.Allowed {
			cpBlock["result"] = "Retrieved"
			if detail.CredentialMaxAge > 0 {
				cpBlock["maxAge"] = detail.CredentialMaxAge
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

	if detail.ToolName != "" {
		payload["toolName"] = detail.ToolName
	}
	if detail.UserID != "" {
		payload["userId"] = detail.UserID
	}

	data, _ := json.Marshal(payload)
	return data
}

func buildCredentialPayload(detail events.CredentialDetail) json.RawMessage {
	outcome := map[string]any{"result": "Unauthorized"}
	if detail.Allowed {
		outcome["result"] = "Authorized"
	}

	cwBlock := map[string]any{"result": "NotFound"}
	if detail.ClientWorkload != "" {
		cwBlock = map[string]any{"name": detail.ClientWorkload, "result": "Identified"}
		if detail.ClientWorkloadID != nil {
			cwBlock["id"] = detail.ClientWorkloadID.String()
		}
	}

	swBlock := map[string]any{"result": "NotFound"}
	if detail.ServerWorkload != "" {
		swBlock = map[string]any{"name": detail.ServerWorkload, "result": "Identified"}
		if detail.ServerWorkloadID != nil {
			swBlock["id"] = detail.ServerWorkloadID.String()
		}
	}

	apBlock := map[string]any{"result": "NotFound"}
	if detail.PolicyName != "" {
		apBlock = map[string]any{"name": detail.PolicyName, "result": "Identified"}
		if detail.PolicyID != nil {
			apBlock["id"] = detail.PolicyID.String()
		}
	}

	tpBlocks := []map[string]any{}
	if len(detail.TrustResults) > 0 {
		for _, tr := range detail.TrustResults {
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
	if detail.CredentialName != "" {
		cpBlock["type"] = detail.CredentialType
		cpBlock["name"] = detail.CredentialName
		if detail.CredentialProviderID != nil {
			cpBlock["id"] = detail.CredentialProviderID.String()
		}
		if detail.Allowed {
			cpBlock["result"] = "Retrieved"
			if detail.CredentialMaxAge > 0 {
				cpBlock["maxAge"] = detail.CredentialMaxAge
			}
		} else {
			cpBlock["result"] = "Failed"
			if detail.DenyReason != "" {
				cpBlock["reason"] = detail.DenyReason
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

func buildDiscoveryPayload(detail events.DiscoveryDetail) json.RawMessage {
	clientWorkloads := []map[string]any{}
	for _, cw := range detail.CandidateClientWorkloads {
		clientWorkloads = append(clientWorkloads, map[string]any{
			"id":   cw.ID.String(),
			"name": cw.Name,
		})
	}

	serverWorkloads := []map[string]any{}
	for _, sw := range detail.CandidateServerWorkloads {
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
