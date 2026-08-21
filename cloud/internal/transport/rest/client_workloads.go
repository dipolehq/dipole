/*
Routes:

	GET    /api/v1/client-workloads
	POST   /api/v1/client-workloads
	GET    /api/v1/client-workloads/{id}
	PUT    /api/v1/client-workloads/{id}
	DELETE /api/v1/client-workloads/{id}
*/
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type clientWorkloadService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ClientWorkload, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClientWorkload, error)
	Create(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error
	Update(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ClientWorkloadHandler struct {
	svc   clientWorkloadService
	audit *events.EventBus
}

func NewClientWorkloadHandler(svc clientWorkloadService) *ClientWorkloadHandler {
	return &ClientWorkloadHandler{svc: svc}
}

func (h *ClientWorkloadHandler) SetAudit(a *events.EventBus) { h.audit = a }

func (h *ClientWorkloadHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/client-workloads", h.List)
	mux.HandleFunc("POST /api/v1/client-workloads", h.Create)
	mux.HandleFunc("GET /api/v1/client-workloads/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/client-workloads/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/client-workloads/{id}", h.Delete)
}

type identifierDTO struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type tagDTO struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type clientWorkloadRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      *bool           `json:"isActive"`
	Identifiers   []identifierDTO `json:"identifiers"`
	Tags          []tagDTO        `json:"tags"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId"`
}

type clientWorkloadResponse struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      bool            `json:"isActive"`
	Identifiers   []identifierDTO `json:"identifiers"`
	Tags          []tagDTO        `json:"tags"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId,omitempty"`
	PolicyCount   int             `json:"policyCount"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (r clientWorkloadRequest) toDomain() domain.ClientWorkload {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	identifiers := make([]domain.Identifier, 0, len(r.Identifiers))
	for _, id := range r.Identifiers {
		identifiers = append(identifiers, domain.Identifier{Type: id.Type, Value: id.Value})
	}

	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}

	return domain.ClientWorkload{
		Name:          r.Name,
		Description:   r.Description,
		IsActive:      isActive,
		Identifiers:   identifiers,
		Tags:          tags,
		ResourceSetID: r.ResourceSetID,
	}
}

func toClientWorkloadResponse(cw *domain.ClientWorkload) clientWorkloadResponse {
	identifiers := make([]identifierDTO, 0, len(cw.Identifiers))
	for _, id := range cw.Identifiers {
		identifiers = append(identifiers, identifierDTO{Type: id.Type, Value: id.Value})
	}

	tags := make([]tagDTO, 0, len(cw.Tags))
	for _, t := range cw.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}

	return clientWorkloadResponse{
		ID:            cw.ID,
		Name:          cw.Name,
		Description:   cw.Description,
		IsActive:      cw.IsActive,
		Identifiers:   identifiers,
		Tags:          tags,
		ResourceSetID: cw.ResourceSetID,
		PolicyCount:   cw.PolicyCount,
		CreatedAt:     cw.CreatedAt,
		UpdatedAt:     cw.UpdatedAt,
	}
}

func (h *ClientWorkloadHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)

	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	resp := make([]clientWorkloadResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toClientWorkloadResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *ClientWorkloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	var req clientWorkloadRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	cw := req.toDomain()
	if err := h.svc.Create(r.Context(), tenantID, &cw); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "client_workload", cw.ID, cw.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toClientWorkloadResponse(&cw))
}

func (h *ClientWorkloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	cw, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toClientWorkloadResponse(cw))
}

func (h *ClientWorkloadHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	var req clientWorkloadRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	cw := req.toDomain()
	cw.ID = id
	if err := h.svc.Update(r.Context(), tenantID, &cw); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "client_workload", cw.ID, cw.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toClientWorkloadResponse(&cw))
}

func (h *ClientWorkloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	if err := h.svc.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "client_workload", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
