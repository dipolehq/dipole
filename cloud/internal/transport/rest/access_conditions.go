/*
Package rest — access_conditions.go is the HTTP handler for the Access
Conditions resource (typed handler → service → repository with DTOs).
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

type accessConditionService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessCondition, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.AccessCondition, error)
	Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error
	Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type AccessConditionHandler struct {
	svc   accessConditionService
	audit *events.EventBus
}

func NewAccessConditionHandler(svc accessConditionService) *AccessConditionHandler {
	return &AccessConditionHandler{svc: svc}
}

func (h *AccessConditionHandler) SetAudit(a *events.EventBus) { h.audit = a }

func (h *AccessConditionHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/access-conditions", h.List)
	mux.HandleFunc("POST /api/v1/access-conditions", h.Create)
	mux.HandleFunc("GET /api/v1/access-conditions/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/access-conditions/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/access-conditions/{id}", h.Delete)
}

type accessConditionRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      *bool           `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Tags          []tagDTO        `json:"tags"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId"`
}

type accessConditionResponse struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      bool            `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Tags          []tagDTO        `json:"tags"`
	PolicyCount   int             `json:"policyCount"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (r accessConditionRequest) toDomain() domain.AccessCondition {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}
	cfg := r.Config
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}
	return domain.AccessCondition{
		Name:          r.Name,
		Description:   r.Description,
		IsActive:      isActive,
		Type:          r.Type,
		Config:        cfg,
		Tags:          tags,
		ResourceSetID: r.ResourceSetID,
	}
}

func toAccessConditionResponse(ac *domain.AccessCondition) accessConditionResponse {
	tags := make([]tagDTO, 0, len(ac.Tags))
	for _, t := range ac.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}
	cfg := ac.Config
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}
	return accessConditionResponse{
		ID:            ac.ID,
		Name:          ac.Name,
		Description:   ac.Description,
		IsActive:      ac.IsActive,
		Type:          ac.Type,
		Config:        cfg,
		Tags:          tags,
		PolicyCount:   ac.PolicyCount,
		ResourceSetID: ac.ResourceSetID,
		CreatedAt:     ac.CreatedAt,
		UpdatedAt:     ac.UpdatedAt,
	}
}

func (h *AccessConditionHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	resp := make([]accessConditionResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toAccessConditionResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *AccessConditionHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req accessConditionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ac := req.toDomain()
	if err := h.svc.Create(r.Context(), tenantID, &ac); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessConditions", "access_condition", ac.ID, ac.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toAccessConditionResponse(&ac))
}

func (h *AccessConditionHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	ac, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if ac == nil {
		respondError(w, http.StatusNotFound, "not_found", "access condition not found")
		return
	}
	respondJSON(w, http.StatusOK, toAccessConditionResponse(ac))
}

func (h *AccessConditionHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	var req accessConditionRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ac := req.toDomain()
	ac.ID = id
	if err := h.svc.Update(r.Context(), tenantID, &ac); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessConditions", "access_condition", ac.ID, ac.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toAccessConditionResponse(&ac))
}

func (h *AccessConditionHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessConditions", "access_condition", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}
