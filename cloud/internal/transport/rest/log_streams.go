package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

// --- Log Stream Handlers ---

func (h *APIHandlers) ListLogStreams(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.logStreams.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list log streams failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toLogStreamResponses(items), total, opts))
}

func (h *APIHandlers) CreateLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if h.planLimiter != nil {
		if err := h.planLimiter.Check(r.Context(), tenantID, service.ResourceLogStreams); err != nil {
			respondBadRequest(w, err.Error())
			return
		}
	}
	var req logStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.Name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	if req.EventType == "" {
		respondBadRequest(w, "eventType is required")
		return
	}
	if req.DestinationType == "" {
		respondBadRequest(w, "destinationType is required")
		return
	}
	validEventTypes := map[string]bool{"access_authorization": true, "audit_logs": true, "workload_events": true}
	if !validEventTypes[req.EventType] {
		respondBadRequest(w, "eventType must be access_authorization, audit_logs, or workload_events")
		return
	}
	validDestTypes := map[string]bool{"aws_s3": true, "gcs_bucket": true, "crowdstrike_siem": true, "splunk_siem": true}
	if !validDestTypes[req.DestinationType] {
		respondBadRequest(w, "destinationType must be aws_s3, gcs_bucket, crowdstrike_siem, or splunk_siem")
		return
	}
	ls := req.toDomain()
	if err := h.logStreams.Create(r.Context(), tenantID, &ls); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.logStreams.GetByID(r.Context(), tenantID, ls.ID)
	if err != nil || saved == nil {
		emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", ls.ID, ls.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusCreated, toLogStreamResponse(&ls))
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toLogStreamResponse(saved))
}

func (h *APIHandlers) GetLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	ls, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if ls == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	respondJSON(w, http.StatusOK, toLogStreamResponse(ls))
}

func (h *APIHandlers) UpdateLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	var req logStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ls := req.toDomain()
	ls.ID = id
	ls.TenantID = tenantID
	if err := h.logStreams.Update(r.Context(), tenantID, &ls); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", ls.ID, ls.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusOK, toLogStreamResponse(&ls))
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toLogStreamResponse(saved))
}

func (h *APIHandlers) DeleteLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	if err := h.logStreams.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

// --- Log Stream DTOs ---

type logStreamRequest struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	IsActive        *bool           `json:"isActive"`
	EventType       string          `json:"eventType"`
	DestinationType string          `json:"destinationType"`
	Config          json.RawMessage `json:"config"`
}

func (r logStreamRequest) toDomain() domain.LogStream {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	config := r.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return domain.LogStream{
		Name:            r.Name,
		Description:     r.Description,
		IsActive:        isActive,
		EventType:       r.EventType,
		DestinationType: r.DestinationType,
		Config:          config,
	}
}

type logStreamResponse struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	IsActive        bool            `json:"isActive"`
	EventType       string          `json:"eventType"`
	DestinationType string          `json:"destinationType"`
	Config          json.RawMessage `json:"config"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// TODO: Encrypt Log Stream config at rest (contains apiKey/authToken for SIEM destinations).
// For now, config is stored as plaintext JSONB — acceptable for MVP behind auth.
func toLogStreamResponse(ls *domain.LogStream) logStreamResponse {
	config := ls.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return logStreamResponse{
		ID:              ls.ID,
		Name:            ls.Name,
		Description:     ls.Description,
		IsActive:        ls.IsActive,
		EventType:       ls.EventType,
		DestinationType: ls.DestinationType,
		Config:          config,
		CreatedAt:       ls.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       ls.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toLogStreamResponses(streams []domain.LogStream) []logStreamResponse {
	resp := make([]logStreamResponse, 0, len(streams))
	for i := range streams {
		resp = append(resp, toLogStreamResponse(&streams[i]))
	}
	return resp
}
