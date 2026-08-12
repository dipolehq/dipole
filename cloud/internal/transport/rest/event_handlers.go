/*
Package rest — event_handlers.go implements the event query API endpoints.

Provides GET endpoints for querying: Access Authorization Events, Workload Events,
and Audit Logs. Supports filtering by timespan, severity, event type, protocol,
and text search. Paginated responses, sorted by timestamp DESC.
*/
package rest

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

type eventQuerier interface {
	QueryAuthEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuthEvent, int, error)
	QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error)
	QueryAuditLogs(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuditLog, int, error)
}

type EventHandlers struct {
	repo eventQuerier
}

func NewEventHandlers(repo eventQuerier) *EventHandlers {
	return &EventHandlers{repo: repo}
}

func (h *EventHandlers) RegisterEventRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/events/authorization", h.QueryAuthorizationEvents)
	mux.HandleFunc("GET /api/v1/events/workload", h.QueryWorkloadEvents)
	mux.HandleFunc("GET /api/v1/events/audit", h.QueryAuditLogs)
}

func (h *EventHandlers) QueryAuthorizationEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseEventQueryOpts(r)

	if et := r.URL.Query().Get("eventType"); et != "" && et != "All" {
		opts.EventType = et
	}
	if src := r.URL.Query().Get("source"); src != "" {
		opts.Source = src
	}

	items, total, err := h.repo.QueryAuthEvents(r.Context(), tenantID, opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":   items,
		"total":   total,
		"page":    opts.Page,
		"perPage": opts.PerPage,
	})
}

func (h *EventHandlers) QueryWorkloadEvents(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseEventQueryOpts(r)

	if proto := r.URL.Query().Get("appProtocol"); proto != "" {
		opts.EventType = proto
	} else if et := r.URL.Query().Get("eventType"); et != "" {
		opts.EventType = et
	}

	// Dedicated workload filters
	var clientWorkloadIDs, serverWorkloadIDs []string
	if cw := r.URL.Query().Get("clientWorkload"); cw != "" {
		clientWorkloadIDs = append(clientWorkloadIDs, cw)
	}
	if sw := r.URL.Query().Get("serverWorkload"); sw != "" {
		serverWorkloadIDs = append(serverWorkloadIDs, sw)
	}
	user := r.URL.Query().Get("user")

	items, total, err := h.repo.QueryWorkloadEvents(r.Context(), tenantID, domain.EventQueryOptions{
		Page:              opts.Page,
		PerPage:           opts.PerPage,
		Search:            opts.Search,
		Severity:          opts.Severity,
		EventType:         opts.EventType,
		StartDate:         opts.StartDate,
		EndDate:           opts.EndDate,
		ClientWorkloadIDs: clientWorkloadIDs,
		ServerWorkloadIDs: serverWorkloadIDs,
		UserID:            user,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":   items,
		"total":   total,
		"page":    opts.Page,
		"perPage": opts.PerPage,
	})
}

func (h *EventHandlers) QueryAuditLogs(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseEventQueryOpts(r)

	if cat := r.URL.Query().Get("category"); cat != "" {
		opts.EventType = cat
	} else if et := r.URL.Query().Get("eventType"); et != "" {
		opts.EventType = et
	}

	items, total, err := h.repo.QueryAuditLogs(r.Context(), tenantID, opts)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":   items,
		"total":   total,
		"page":    opts.Page,
		"perPage": opts.PerPage,
	})
}

func parseEventQueryOpts(r *http.Request) domain.EventQueryOptions {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("perPage"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 100
	}

	opts := domain.EventQueryOptions{
		Page:    page,
		PerPage: perPage,
		Search:  q.Get("search"),
	}

	if sev := q.Get("severity"); sev != "" && sev != "All" {
		opts.Severity = sev
	}

	now := time.Now().UTC()
	if spanHours := q.Get("spanLastHours"); spanHours != "" {
		hours, _ := strconv.Atoi(spanHours)
		if hours > 0 {
			start := now.Add(-time.Duration(hours) * time.Hour)
			opts.StartDate = &start
			opts.EndDate = &now
		}
	} else if spanDays := q.Get("spanLastDays"); spanDays != "" {
		days, _ := strconv.Atoi(spanDays)
		if days > 0 {
			start := now.AddDate(0, 0, -days)
			opts.StartDate = &start
			opts.EndDate = &now
		}
	}

	return opts
}
