/*
Routes:

	GET    /api/v1/log-streams
	POST   /api/v1/log-streams
	GET    /api/v1/log-streams/{id}
	PUT    /api/v1/log-streams/{id}
	DELETE /api/v1/log-streams/{id}
*/
package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

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
	var req logStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ls := req.toDomain()
	if err := h.logStreams.Create(r.Context(), tenantID, &ls); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.logStreams.Get(r.Context(), tenantID, ls.ID)
	if err != nil {
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
	ls, err := h.logStreams.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
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
	saved, err := h.logStreams.Get(r.Context(), tenantID, id)
	if err != nil {
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
	if err := h.logStreams.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "LogStreams", "log_stream", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

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
	LastExportedAt  *string         `json:"lastExportedAt,omitempty"`
	LastExportError *string         `json:"lastExportError,omitempty"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

func toLogStreamResponse(ls *domain.LogStream) logStreamResponse {
	config := redactLogStreamConfig(ls.Config)
	var lastExportedAt *string
	if ls.LastExportedAt != nil {
		formatted := ls.LastExportedAt.Format("2006-01-02T15:04:05Z07:00")
		lastExportedAt = &formatted
	}
	return logStreamResponse{
		ID:              ls.ID,
		Name:            ls.Name,
		Description:     ls.Description,
		IsActive:        ls.IsActive,
		EventType:       ls.EventType,
		DestinationType: ls.DestinationType,
		Config:          config,
		LastExportedAt:  lastExportedAt,
		LastExportError: ls.LastExportError,
		CreatedAt:       ls.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       ls.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

var logStreamSensitiveConfigKeys = map[string]bool{
	"hec_token":            true,
	"client_secret":        true,
	"access_key_id":        true,
	"secret_access_key":    true,
	"service_account_json": true,
}

func redactLogStreamConfig(config json.RawMessage) json.RawMessage {
	if len(config) == 0 {
		return json.RawMessage(`{}`)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(config, &parsed); err != nil {
		return json.RawMessage(`{}`)
	}

	for key := range parsed {
		if logStreamSensitiveConfigKeys[key] {
			parsed[key] = "**REDACTED**"
		}
	}

	redacted, err := json.Marshal(parsed)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return redacted
}

func toLogStreamResponses(streams []domain.LogStream) []logStreamResponse {
	resp := make([]logStreamResponse, 0, len(streams))
	for i := range streams {
		resp = append(resp, toLogStreamResponse(&streams[i]))
	}
	return resp
}
