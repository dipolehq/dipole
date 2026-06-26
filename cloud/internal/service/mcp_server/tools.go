/*
Package mcp_server implements the Dipole MCP Server business logic.

tools.go defines the three read-only MCP tools (get_audit_logs, get_auth_events,
get_workload_events) with their parameter schemas. It also implements the query
logic with date range priority, pagination capping (max 100), and filter mapping.
*/
package mcp_server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

const maxPerPage = 100

type ToolService struct {
	eventRepo   repository.EventRepository
	auditLogger *AuditLogger
}

func NewToolService(eventRepo repository.EventRepository, auditLogger *AuditLogger) *ToolService {
	return &ToolService{eventRepo: eventRepo, auditLogger: auditLogger}
}

func (s *ToolService) ListTools() any {
	return map[string]any{"tools": toolDefinitions}
}

func (s *ToolService) CallTool(name string, args map[string]any, tenantID uuid.UUID, resourceSetID *uuid.UUID) (any, error) {
	if s.auditLogger != nil {
		s.auditLogger.LogToolCall(context.Background(), tenantID, name, "")
	}

	switch name {
	case "get_audit_logs":
		return s.getAuditLogs(args, tenantID)
	case "get_auth_events":
		return s.getAuthEvents(args, tenantID)
	case "get_workload_events":
		return s.getWorkloadEvents(args, tenantID)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

func (s *ToolService) getAuditLogs(args map[string]any, tenantID uuid.UUID) (any, error) {
	opts := buildAuditLogOpts(args)
	logs, total, err := s.eventRepo.QueryAuditLogs(context.Background(), tenantID, opts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": logs, "total": total, "page": opts.Page, "perPage": opts.PerPage}, nil
}

func (s *ToolService) getAuthEvents(args map[string]any, tenantID uuid.UUID) (any, error) {
	opts := buildAuthEventOpts(args)
	events, total, err := s.eventRepo.QueryAuthEvents(context.Background(), tenantID, opts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": events, "total": total, "page": opts.Page, "perPage": opts.PerPage}, nil
}

func (s *ToolService) getWorkloadEvents(args map[string]any, tenantID uuid.UUID) (any, error) {
	opts := buildWorkloadEventOpts(args)
	events, total, err := s.eventRepo.QueryWorkloadEvents(context.Background(), tenantID, opts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"items": events, "total": total, "page": opts.Page, "perPage": opts.PerPage}, nil
}

func buildAuditLogOpts(args map[string]any) domain.EventQueryOptions {
	opts := domain.EventQueryOptions{
		Page:    intArg(args, "page", 1),
		PerPage: capPerPage(intArg(args, "perPage", 100)),
		OrderBy: strArg(args, "orderBy", "CreatedAt"),
		Search:  strArg(args, "search", ""),
	}

	opts.Severity = strArg(args, "severity", "")
	opts.EventType = strArg(args, "category", "")

	startDate, endDate := resolveAuditDateRange(args)
	opts.StartDate = startDate
	opts.EndDate = endDate

	return opts
}

func buildAuthEventOpts(args map[string]any) domain.EventQueryOptions {
	opts := domain.EventQueryOptions{
		Page:    intArg(args, "page", 1),
		PerPage: capPerPage(intArg(args, "perPage", 100)),
		OrderBy: strArg(args, "orderBy", "Timestamp"),
		Search:  strArg(args, "search", ""),
	}

	opts.Severity = strArg(args, "severity", "")

	if et := strArg(args, "eventType", ""); et != "" {
		opts.EventType = mapEventType(et)
	}

	startDate, endDate := resolveAuthDateRange(args)
	opts.StartDate = startDate
	opts.EndDate = endDate

	return opts
}

func buildWorkloadEventOpts(args map[string]any) domain.EventQueryOptions {
	opts := domain.EventQueryOptions{
		Page:    intArg(args, "page", 1),
		PerPage: capPerPage(intArg(args, "perPage", 100)),
		OrderBy: strArg(args, "orderBy", "Timestamp"),
		Search:  strArg(args, "search", ""),
	}

	opts.Severity = strArg(args, "severity", "")
	opts.EventType = strArg(args, "appProtocol", "")

	startDate, endDate := resolveAuthDateRange(args)
	opts.StartDate = startDate
	opts.EndDate = endDate

	return opts
}

func resolveAuditDateRange(args map[string]any) (*time.Time, *time.Time) {
	now := time.Now().UTC()

	if sd := strArg(args, "startDate", ""); sd != "" {
		if ed := strArg(args, "endDate", ""); ed != "" {
			startDate, _ := time.Parse(time.RFC3339, sd)
			endDate, _ := time.Parse(time.RFC3339, ed)
			endDate = clampToNow(endDate, now)
			return &startDate, &endDate
		}
	}

	if mins := intArg(args, "spanLastMinutes", 0); mins > 0 {
		start := now.Add(-time.Duration(mins) * time.Minute)
		return &start, &now
	}

	days := intArg(args, "spanLastDays", 30)
	start := now.AddDate(0, 0, -days)
	return &start, &now
}

func resolveAuthDateRange(args map[string]any) (*time.Time, *time.Time) {
	now := time.Now().UTC()

	if sd := strArg(args, "startDate", ""); sd != "" {
		if ed := strArg(args, "endDate", ""); ed != "" {
			startDate, _ := time.Parse(time.RFC3339, sd)
			endDate, _ := time.Parse(time.RFC3339, ed)
			endDate = clampToNow(endDate, now)
			return &startDate, &endDate
		}
	}

	if mins := intArg(args, "spanLastMinutes", 0); mins > 0 {
		start := now.Add(-time.Duration(mins) * time.Minute)
		return &start, &now
	}

	hours := intArg(args, "spanLastHours", 24)
	start := now.Add(-time.Duration(hours) * time.Hour)
	return &start, &now
}

func mapEventType(et string) string {
	switch et {
	case "Request":
		return "access.request"
	case "Authorization":
		return "access.authorization"
	case "Credential":
		return "access.credential"
	default:
		return et
	}
}

func clampToNow(endDate, now time.Time) time.Time {
	if endDate.After(now) {
		return now
	}
	return endDate
}

func capPerPage(v int) int {
	if v > maxPerPage {
		return maxPerPage
	}
	if v < 1 {
		return 25
	}
	return v
}

func intArg(args map[string]any, key string, defaultVal int) int {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return defaultVal
	}
}

func strArg(args map[string]any, key string, defaultVal string) string {
	v, ok := args[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}
