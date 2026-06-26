/*
audit.go logs every MCP tool call to the audit trail for compliance.

AuditLogger records WHO queried WHAT via AI agents, with category "Analysis"
and severity "Info". This ensures there's a trail of AI-driven data access.
*/
package mcp_server

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type AuditLogger struct {
	repo repository.EventRepository
}

func NewAuditLogger(repo repository.EventRepository) *AuditLogger {
	return &AuditLogger{repo: repo}
}

func (l *AuditLogger) LogToolCall(ctx context.Context, tenantID uuid.UUID, toolName string, actorEmail string) {
	if l.repo == nil {
		return
	}

	activity := mapToolToActivity(toolName)

	l.repo.InsertAuditLog(ctx, tenantID, &domain.AuditLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ActorEmail: actorEmail,
		Category:   "Analysis",
		Activity:   activity,
		TargetType: "mcp_server_query",
		TargetName: toolName,
		Severity:   "Info",
		Timestamp:  time.Now().UTC(),
	})
}

func mapToolToActivity(toolName string) string {
	switch toolName {
	case "get_audit_logs":
		return "QueryAuditLogs"
	case "get_auth_events":
		return "QueryAuthEvents"
	case "get_workload_events":
		return "QueryWorkloadEvents"
	default:
		return "QueryUnknown"
	}
}
