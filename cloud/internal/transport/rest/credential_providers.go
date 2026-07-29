// Package rest — credential_providers.go is the HTTP handler for the
// Credential Providers resource with encrypted config at rest.
package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type credentialProviderService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.CredentialProvider, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.CredentialProvider, json.RawMessage, error)
	Create(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider, configJSON json.RawMessage) error
	Update(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider, configJSON json.RawMessage) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type CredentialProviderHandler struct {
	svc credentialProviderService
}

func NewCredentialProviderHandler(svc credentialProviderService) *CredentialProviderHandler {
	return &CredentialProviderHandler{svc: svc}
}

func (h *CredentialProviderHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/credential-providers", h.List)
	mux.HandleFunc("POST /api/v1/credential-providers", h.Create)
	mux.HandleFunc("GET /api/v1/credential-providers/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/credential-providers/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/credential-providers/{id}", h.Delete)
}

type credentialProviderRequest struct {
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      *bool           `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Tags          []tagDTO        `json:"tags"`
	IntegrationID *uuid.UUID      `json:"integrationId"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId"`
}

type credentialProviderListResponse struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	IsActive      bool       `json:"isActive"`
	Type          string     `json:"type"`
	Tags          []tagDTO   `json:"tags"`
	PolicyCount   int        `json:"policyCount"`
	ResourceSetID *uuid.UUID `json:"resourceSetId,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

type credentialProviderDetailResponse struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	IsActive      bool            `json:"isActive"`
	Type          string          `json:"type"`
	Config        json.RawMessage `json:"config"`
	Tags          []tagDTO        `json:"tags"`
	IntegrationID *uuid.UUID      `json:"integrationId,omitempty"`
	PolicyCount   int             `json:"policyCount"`
	ResourceSetID *uuid.UUID      `json:"resourceSetId,omitempty"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (r credentialProviderRequest) toDomain() domain.CredentialProvider {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}
	return domain.CredentialProvider{
		Name:          r.Name,
		Description:   r.Description,
		IsActive:      isActive,
		Type:          r.Type,
		Tags:          tags,
		IntegrationID: r.IntegrationID,
		ResourceSetID: r.ResourceSetID,
	}
}

func toCredentialProviderListResponse(cp *domain.CredentialProvider) credentialProviderListResponse {
	tags := make([]tagDTO, 0, len(cp.Tags))
	for _, t := range cp.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}
	return credentialProviderListResponse{
		ID:            cp.ID,
		Name:          cp.Name,
		Description:   cp.Description,
		IsActive:      cp.IsActive,
		Type:          cp.Type,
		Tags:          tags,
		PolicyCount:   cp.PolicyCount,
		ResourceSetID: cp.ResourceSetID,
		CreatedAt:     cp.CreatedAt,
		UpdatedAt:     cp.UpdatedAt,
	}
}

func toCredentialProviderDetailResponse(cp *domain.CredentialProvider, config json.RawMessage) credentialProviderDetailResponse {
	tags := make([]tagDTO, 0, len(cp.Tags))
	for _, t := range cp.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}
	if config == nil {
		config = json.RawMessage("{}")
	}
	return credentialProviderDetailResponse{
		ID:            cp.ID,
		Name:          cp.Name,
		Description:   cp.Description,
		IsActive:      cp.IsActive,
		Type:          cp.Type,
		Config:        config,
		Tags:          tags,
		IntegrationID: cp.IntegrationID,
		PolicyCount:   cp.PolicyCount,
		ResourceSetID: cp.ResourceSetID,
		CreatedAt:     cp.CreatedAt,
		UpdatedAt:     cp.UpdatedAt,
	}
}

func (h *CredentialProviderHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	resp := make([]credentialProviderListResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toCredentialProviderListResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *CredentialProviderHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req credentialProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	cp := req.toDomain()
	configJSON := req.Config
	if configJSON == nil {
		configJSON = json.RawMessage("{}")
	}
	if err := h.svc.Create(r.Context(), tenantID, &cp, configJSON); err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toCredentialProviderDetailResponse(&cp, configJSON))
}

func (h *CredentialProviderHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	cp, config, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toCredentialProviderDetailResponse(cp, redactCredentialConfig(config)))
}

func (h *CredentialProviderHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	var req credentialProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	cp := req.toDomain()
	cp.ID = id
	configJSON := req.Config
	if configJSON == nil {
		configJSON = json.RawMessage("{}")
	}
	if err := h.svc.Update(r.Context(), tenantID, &cp, configJSON); err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toCredentialProviderDetailResponse(&cp, configJSON))
}

func (h *CredentialProviderHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	w.WriteHeader(http.StatusNoContent)
}

// redactCredentialConfig removes sensitive values from credential provider
// config before returning in API responses. Non-sensitive fields (URLs,
// scopes, audience, subject) are preserved for the UI to display.
func redactCredentialConfig(config json.RawMessage) json.RawMessage {
	if config == nil || len(config) == 0 {
		return json.RawMessage("{}")
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(config, &parsed); err != nil {
		return json.RawMessage("{}")
	}

	sensitiveKeys := map[string]bool{
		"api_key":              true,
		"password":             true,
		"client_secret":        true,
		"private_key":          true,
		"secret_key":           true,
		"signing_key":          true,
		"service_account_json": true,
		"hec_token":            true,
		"access_key":           true,
		"refresh_token":        true,
	}

	for key := range parsed {
		if sensitiveKeys[key] {
			parsed[key] = "**REDACTED**"
		}
	}

	redacted, _ := json.Marshal(parsed)
	return redacted
}
