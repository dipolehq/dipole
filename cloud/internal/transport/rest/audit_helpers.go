// audit_helpers.go provides shared audit event emission helpers and client IP/user-agent extraction for HTTP handlers.
package rest

import (
	"context"
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/google/uuid"
)

const (
	auditSeverityInfo  = "Info"
	auditSeverityWarn  = "Warn"
	auditSeverityAlert = "Alert"
)

// emitAudit builds and emits an AuditEvent, attributing it to the
// resource set active in ctx (see scope.FromContext) and, when the caller
// is a real admin user rather than an API token identity, to that user's
// ID (see UserIDFromContext) in addition to their email.
func emitAudit(bus *events.EventBus, ctx context.Context, tenantID uuid.UUID, actorEmail string, category string, activity string, severity string, targetType string, targetID uuid.UUID, targetName string, ip string, ua string) {
	if bus == nil {
		return
	}
	browser, os := parseUA(ua)

	var resourceSetID *uuid.UUID
	if s := scope.FromContext(ctx); s != nil {
		resourceSetID = &s.ResourceSetID
	}

	bus.EmitAudit(ctx, events.AuditEvent{
		TenantID:      tenantID,
		ActorID:       UserIDFromContext(ctx),
		ActorEmail:    actorEmail,
		ActorIP:       ip,
		Browser:       browser,
		OS:            os,
		Category:      category,
		Activity:      activity,
		Severity:      severity,
		TargetType:    targetType,
		TargetID:      &targetID,
		TargetName:    targetName,
		ResourceSetID: resourceSetID,
	})
}

func emitAuditCreated(bus *events.EventBus, ctx context.Context, tenantID uuid.UUID, actorEmail string, category string, targetType string, targetID uuid.UUID, targetName string, ip string, ua string) {
	emitAudit(bus, ctx, tenantID, actorEmail, category, "Created", auditSeverityInfo, targetType, targetID, targetName, ip, ua)
}

func emitAuditUpdated(bus *events.EventBus, ctx context.Context, tenantID uuid.UUID, actorEmail string, category string, targetType string, targetID uuid.UUID, targetName string, ip string, ua string) {
	emitAudit(bus, ctx, tenantID, actorEmail, category, "Updated", auditSeverityInfo, targetType, targetID, targetName, ip, ua)
}

func emitAuditDeleted(bus *events.EventBus, ctx context.Context, tenantID uuid.UUID, actorEmail string, category string, targetType string, targetID uuid.UUID, targetName string, ip string, ua string) {
	emitAudit(bus, ctx, tenantID, actorEmail, category, "Deleted", auditSeverityWarn, targetType, targetID, targetName, ip, ua)
}

func clientIP(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.SplitN(forwarded, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

func userAgent(r *http.Request) string {
	return r.Header.Get("User-Agent")
}

func parseUA(ua string) (browser, os string) {
	if ua == "" {
		return "", ""
	}
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
