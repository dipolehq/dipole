/*
Package rest — trust_providers.go is the HTTP handler for the Trust
Providers resource (typed handler → service → repository with DTOs).
*/
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type trustProviderService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.TrustProvider, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.TrustProvider, error)
	Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error
	Update(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type TrustProviderHandler struct {
	svc   trustProviderService
	audit *AuditEmitter
}

func NewTrustProviderHandler(svc trustProviderService) *TrustProviderHandler {
	return &TrustProviderHandler{svc: svc}
}

func (h *TrustProviderHandler) SetAudit(a *AuditEmitter) { h.audit = a }

func (h *TrustProviderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/trust-providers", h.List)
	mux.HandleFunc("POST /api/v1/trust-providers", h.Create)
	mux.HandleFunc("GET /api/v1/trust-providers/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/trust-providers/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/trust-providers/{id}", h.Delete)
}

type matchRuleDTO struct {
	Attribute string `json:"attribute"`
	Value     string `json:"value"`
}

type trustProviderRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      *bool           `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	MatchRules    []matchRuleDTO  `json:"matchRules"`
	Tags          []tagDTO        `json:"tags"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId"`
}

type trustProviderResponse struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	IsActive        bool            `json:"isActive"`
	Type            string          `json:"type"`
	Config          json.RawMessage `json:"config"`
	MatchRules      []matchRuleDTO  `json:"matchRules"`
	EdgeSDKClientID string          `json:"edgeSdkClientId"`
	EdgeSDKAudience string          `json:"edgeSdkAudience"`
	Tags            []tagDTO        `json:"tags"`
	PolicyCount     int             `json:"policyCount"`
	ResourceSetID   *uuid.UUID      `json:"resourceSetId,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

func (r trustProviderRequest) toDomain() domain.TrustProvider {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	matchRules := make([]domain.MatchRule, 0, len(r.MatchRules))
	for _, mr := range r.MatchRules {
		if mr.Attribute != "" {
			matchRules = append(matchRules, domain.MatchRule{Attribute: mr.Attribute, Value: mr.Value})
		}
	}

	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}

	cfg := r.Config
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}

	return domain.TrustProvider{
		Name:          r.Name,
		Description:   r.Description,
		IsActive:      isActive,
		Type:          r.Type,
		Config:        cfg,
		MatchRules:    matchRules,
		Tags:          tags,
		ResourceSetID: r.ResourceSetID,
	}
}

func toTrustProviderResponse(tp *domain.TrustProvider) trustProviderResponse {
	matchRules := make([]matchRuleDTO, 0, len(tp.MatchRules))
	for _, mr := range tp.MatchRules {
		matchRules = append(matchRules, matchRuleDTO{Attribute: mr.Attribute, Value: mr.Value})
	}

	tags := make([]tagDTO, 0, len(tp.Tags))
	for _, t := range tp.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}

	cfg := tp.Config
	if cfg == nil {
		cfg = json.RawMessage("{}")
	}

	return trustProviderResponse{
		ID:              tp.ID,
		Name:            tp.Name,
		Description:     tp.Description,
		IsActive:        tp.IsActive,
		Type:            tp.Type,
		Config:          cfg,
		MatchRules:      matchRules,
		EdgeSDKClientID: tp.EdgeSDKClientID,
		EdgeSDKAudience: tp.EdgeSDKAudience,
		Tags:            tags,
		PolicyCount:     tp.PolicyCount,
		ResourceSetID:   tp.ResourceSetID,
		CreatedAt:       tp.CreatedAt,
		UpdatedAt:       tp.UpdatedAt,
	}
}

func (h *TrustProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)

	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	resp := make([]trustProviderResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toTrustProviderResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *TrustProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	var req trustProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	tp := req.toDomain()
	if err := h.svc.Create(r.Context(), tenantID, &tp); err != nil {
		respondDomainError(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Created(r.Context(), tenantID, UserEmailFromContext(r.Context()), "TrustProvider", "trust_provider", tp.ID, tp.Name, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusCreated, toTrustProviderResponse(&tp))
}

func (h *TrustProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	tp, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toTrustProviderResponse(tp))
}

func (h *TrustProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	var req trustProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	tp := req.toDomain()
	tp.ID = id
	if err := h.svc.Update(r.Context(), tenantID, &tp); err != nil {
		respondDomainError(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "TrustProvider", "trust_provider", tp.ID, tp.Name, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, toTrustProviderResponse(&tp))
}

func (h *TrustProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	if h.audit != nil {
		h.audit.Deleted(r.Context(), tenantID, UserEmailFromContext(r.Context()), "TrustProvider", "trust_provider", id, "", clientIP(r), userAgent(r))
	}
	w.WriteHeader(http.StatusNoContent)
}
