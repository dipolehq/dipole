/*
Package events implements event emission for the Dipole access control system.

auth_events.go emits Access Authorization Events (access.request, access.authorization,
access.credential) linked by a shared contextId UUID. Called at the end of every
policy evaluation to create a complete audit trail of access decisions.
*/
package events

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type AuthEventEmitter struct {
	repo repository.EventRepository
}

func NewAuthEventEmitter(repo repository.EventRepository) *AuthEventEmitter {
	return &AuthEventEmitter{repo: repo}
}

type EvaluationContext struct {
	TenantID                 uuid.UUID
	SourceIP                 string
	TargetHost               string
	TargetPort               int
	ResourceSetID            *uuid.UUID
	Result                   *domain.EvaluationResult
	ClientWorkload           *domain.ClientWorkload
	ServerWorkload           *domain.ServerWorkload
	Policy                   *domain.AccessPolicy
	TrustProviders           []domain.TrustProvider
	CredentialProvider       *domain.CredentialProvider
	CandidateClientWorkloads []domain.ClientWorkload
	CandidateServerWorkloads []domain.ServerWorkload
}

func (e *AuthEventEmitter) Emit(ctx context.Context, eval *EvaluationContext) error {
	contextID := uuid.New()
	now := time.Now().UTC()

	severity := determineSeverity(eval.Result)

	// access.request
	requestPayload := buildRequestPayload(eval)
	if err := e.repo.InsertAuthEvent(ctx, eval.TenantID, &domain.AuthEvent{
		ID:            uuid.New(),
		TenantID:      eval.TenantID,
		EventType:     domain.EventTypeAccessRequest,
		ContextID:     contextID,
		Severity:      domain.SeverityInfo,
		ClientIP:      eval.SourceIP,
		Timestamp:     now,
		Payload:       requestPayload,
		ResourceSetID: eval.ResourceSetID,
	}); err != nil {
		return err
	}

	// access.authorization
	authPayload := buildAuthorizationPayload(eval)
	if err := e.repo.InsertAuthEvent(ctx, eval.TenantID, &domain.AuthEvent{
		ID:            uuid.New(),
		TenantID:      eval.TenantID,
		EventType:     domain.EventTypeAccessAuthorization,
		ContextID:     contextID,
		Severity:      severity,
		ClientIP:      eval.SourceIP,
		Timestamp:     now,
		Payload:       authPayload,
		ResourceSetID: eval.ResourceSetID,
	}); err != nil {
		return err
	}

	// access.credential (if credential was involved)
	if eval.CredentialProvider != nil {
		credPayload := buildCredentialPayload(eval)
		credSeverity := domain.SeverityInfo
		if eval.Result != nil && !eval.Result.Allowed {
			credSeverity = domain.SeverityError
		}
		if err := e.repo.InsertAuthEvent(ctx, eval.TenantID, &domain.AuthEvent{
			ID:            uuid.New(),
			TenantID:      eval.TenantID,
			EventType:     domain.EventTypeAccessCredential,
			ContextID:     contextID,
			Severity:      credSeverity,
			ClientIP:      eval.SourceIP,
			Timestamp:     now,
			Payload:       credPayload,
			ResourceSetID: eval.ResourceSetID,
		}); err != nil {
			return err
		}
	}

	// access.discovery (lists candidate workloads considered during evaluation)
	discoveryPayload := buildDiscoveryPayload(eval)
	discoverySeverity := domain.SeverityInfo
	if eval.Result != nil && !eval.Result.Allowed && eval.ClientWorkload == nil {
		discoverySeverity = domain.SeverityError
	}
	if err := e.repo.InsertAuthEvent(ctx, eval.TenantID, &domain.AuthEvent{
		ID:            uuid.New(),
		TenantID:      eval.TenantID,
		EventType:     domain.EventTypeAccessDiscovery,
		ContextID:     contextID,
		Severity:      discoverySeverity,
		ClientIP:      eval.SourceIP,
		Timestamp:     now,
		Payload:       discoveryPayload,
		ResourceSetID: eval.ResourceSetID,
	}); err != nil {
		return err
	}

	return nil
}

func determineSeverity(result *domain.EvaluationResult) string {
	if result == nil {
		return domain.SeverityError
	}
	if result.Allowed {
		return domain.SeverityInfo
	}
	if result.DenyReason == "InternalError" {
		return domain.SeverityError
	}
	return domain.SeverityWarn
}

func buildRequestPayload(eval *EvaluationContext) json.RawMessage {
	payload := map[string]any{
		"clientRequest": map[string]any{
			"version": "1.0.0",
			"network": map[string]any{
				"sourceIP":          eval.SourceIP,
				"sourcePort":        0,
				"transportProtocol": "TCP",
				"proxyPort":         8080,
				"targetHost":        eval.TargetHost,
				"targetPort":        eval.TargetPort,
			},
		},
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildAuthorizationPayload(eval *EvaluationContext) json.RawMessage {
	outcome := map[string]any{"result": "Error"}
	if eval.Result != nil {
		if eval.Result.Allowed {
			outcome["result"] = "Authorized"
		} else {
			outcome["result"] = "Unauthorized"
			outcome["reason"] = eval.Result.DenyReason
		}
	}

	cwBlock := map[string]any{"result": "NotFound"}
	if eval.ClientWorkload != nil {
		cwBlock = map[string]any{"id": eval.ClientWorkload.ID.String(), "name": eval.ClientWorkload.Name, "result": "Identified"}
	}

	swBlock := map[string]any{"result": "NotFound"}
	if eval.ServerWorkload != nil {
		swBlock = map[string]any{"id": eval.ServerWorkload.ID.String(), "name": eval.ServerWorkload.Name, "result": "Identified"}
	}

	apBlock := map[string]any{"result": "NotFound"}
	if eval.Policy != nil {
		apBlock = map[string]any{"id": eval.Policy.ID.String(), "name": eval.Policy.Name, "result": "Identified"}
	}

	tpBlocks := []map[string]any{}
	if eval.Result != nil {
		for _, tr := range eval.Result.TrustResults {
			tp := map[string]any{
				"id":   tr.ProviderID.String(),
				"name": tr.ProviderName,
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
	}

	cpBlock := map[string]any{}
	if eval.CredentialProvider != nil {
		cpBlock = map[string]any{
			"id":   eval.CredentialProvider.ID.String(),
			"name": eval.CredentialProvider.Name,
			"type": eval.CredentialProvider.Type,
		}
		if eval.Result != nil && eval.Result.Allowed && eval.Result.Credential != nil {
			cpBlock["result"] = "Retrieved"
			cpBlock["maxAge"] = 60
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
	data, _ := json.Marshal(payload)
	return data
}

func buildCredentialPayload(eval *EvaluationContext) json.RawMessage {
	outcome := map[string]any{"result": "Unauthorized"}
	if eval.Result != nil && eval.Result.Allowed {
		outcome["result"] = "Authorized"
	}

	cpBlock := map[string]any{
		"type": eval.CredentialProvider.Type,
		"id":   eval.CredentialProvider.ID.String(),
		"name": eval.CredentialProvider.Name,
	}
	if eval.Result != nil && eval.Result.Allowed && eval.Result.Credential != nil {
		cpBlock["result"] = "Retrieved"
		cpBlock["maxAge"] = 60
	} else {
		cpBlock["result"] = "Failed"
		if eval.Result != nil {
			cpBlock["reason"] = eval.Result.DenyReason
		}
	}

	payload := map[string]any{
		"outcome":            outcome,
		"credentialProvider": cpBlock,
	}
	data, _ := json.Marshal(payload)
	return data
}

func buildDiscoveryPayload(eval *EvaluationContext) json.RawMessage {
	clientWorkloads := []map[string]any{}
	for _, cw := range eval.CandidateClientWorkloads {
		clientWorkloads = append(clientWorkloads, map[string]any{
			"id":   cw.ID.String(),
			"name": cw.Name,
		})
	}

	serverWorkloads := []map[string]any{}
	for _, sw := range eval.CandidateServerWorkloads {
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
