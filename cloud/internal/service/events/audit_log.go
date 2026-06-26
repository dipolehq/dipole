/*
audit_log.go emits audit log entries for every successful API mutation.

AuditLogEmitter records WHO changed WHAT (create/update/delete) on which
resource. Only successful mutations are logged — validation errors and
not-found responses do NOT generate audit entries.
*/
package events

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type AuditLogEmitter struct {
	repo repository.EventRepository
}

func NewAuditLogEmitter(repo repository.EventRepository) *AuditLogEmitter {
	return &AuditLogEmitter{repo: repo}
}

func (e *AuditLogEmitter) EmitAuditLog(ctx context.Context, tenantID uuid.UUID, actorEmail, category, activity, targetType string, targetID uuid.UUID, targetName, clientIP, userAgent string) {
	if e.repo == nil {
		return
	}

	browser, os := parseUserAgent(userAgent)

	e.repo.InsertAuditLog(ctx, tenantID, &domain.AuditLog{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ActorEmail: actorEmail,
		Category:   category,
		Activity:   activity,
		TargetType: targetType,
		TargetID:   &targetID,
		TargetName: targetName,
		Severity:   domain.SeverityInfo,
		ClientIP:   clientIP,
		Timestamp:  time.Now().UTC(),
		Result:     "Success",
		Browser:    browser,
		OS:         os,
	})
}

func (e *AuditLogEmitter) Created(ctx context.Context, tenantID uuid.UUID, actorEmail, category, targetType string, targetID uuid.UUID, targetName, clientIP, userAgent string) {
	e.EmitAuditLog(ctx, tenantID, actorEmail, category, "Created", targetType, targetID, targetName, clientIP, userAgent)
}

func (e *AuditLogEmitter) Updated(ctx context.Context, tenantID uuid.UUID, actorEmail, category, targetType string, targetID uuid.UUID, targetName, clientIP, userAgent string) {
	e.EmitAuditLog(ctx, tenantID, actorEmail, category, "Updated", targetType, targetID, targetName, clientIP, userAgent)
}

func (e *AuditLogEmitter) Deleted(ctx context.Context, tenantID uuid.UUID, actorEmail, category, targetType string, targetID uuid.UUID, targetName, clientIP, userAgent string) {
	e.EmitAuditLog(ctx, tenantID, actorEmail, category, "Deleted", targetType, targetID, targetName, clientIP, userAgent)
}

func parseUserAgent(ua string) (browser, os string) {
	if ua == "" {
		return "", ""
	}
	// Simple parsing — extract browser and OS from common patterns
	switch {
	case strings.Contains(ua, "Edge") || strings.Contains(ua, "Edg/"):
		browser = "Edge"
	case strings.Contains(ua, "Chrome"):
		browser = "Chrome"
	case strings.Contains(ua, "Firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "Safari"):
		browser = "Safari"
	default:
		browser = "Other"
	}
	switch {
	case strings.Contains(ua, "Windows"):
		os = "Windows"
	case strings.Contains(ua, "Mac OS"):
		os = "macOS"
	case strings.Contains(ua, "Linux"):
		os = "Linux"
	case strings.Contains(ua, "Android"):
		os = "Android"
	case strings.Contains(ua, "iOS"):
		os = "iOS"
	default:
		os = "Other"
	}
	return
}

// CategoryForEntity maps entity types to audit log categories.
func CategoryForEntity(entityType string) string {
	switch entityType {
	case "client_workload", "server_workload":
		return "Workloads"
	case "access_policy":
		return "AccessPolicies"
	case "trust_provider":
		return "TrustProvider"
	case "credential_provider":
		return "CredentialProvider"
	case "agent_controller":
		return "AgentControllers"
	case "access_condition":
		return "AccessConditions"
	case "role":
		return "Roles"
	case "user":
		return "Users"
	case "resource_set":
		return "ResourceSets"
	case "log_stream":
		return "LogStreams"
	case "identity_provider":
		return "IdentityProviders"
	case "sign_on_policy":
		return "SignOnPolicies"
	default:
		return "Unknown"
	}
}
