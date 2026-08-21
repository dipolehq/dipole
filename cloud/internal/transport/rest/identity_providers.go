/*
Routes:

	GET    /api/v1/identity-providers
	POST   /api/v1/identity-providers
	GET    /api/v1/identity-providers/{id}
	PUT    /api/v1/identity-providers/{id}
	DELETE /api/v1/identity-providers/{id}
*/
package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

func (h *APIHandlers) ListIdentityProviders(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.identityProviders.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list identity providers failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toIdpResponses(items), total, opts))
}

func (h *APIHandlers) CreateIdentityProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req idpRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	idp := req.toDomain()
	if err := h.identityProviders.Create(r.Context(), tenantID, &idp); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.identityProviders.Get(r.Context(), tenantID, idp.ID)
	if err != nil {
		emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", idp.ID, idp.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusCreated, toIdpResponse(&idp))
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toIdpResponse(saved))
}

func (h *APIHandlers) GetIdentityProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	idp, err := h.identityProviders.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toIdpResponse(idp))
}

func (h *APIHandlers) UpdateIdentityProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	var req idpRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	idp := req.toDomain()
	idp.ID = id
	idp.TenantID = tenantID
	if err := h.identityProviders.Update(r.Context(), tenantID, &idp); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.identityProviders.Get(r.Context(), tenantID, id)
	if err != nil {
		emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", idp.ID, idp.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusOK, toIdpResponse(&idp))
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toIdpResponse(saved))
}

func (h *APIHandlers) DeleteIdentityProvider(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	if err := h.identityProviders.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

type idpRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    *bool           `json:"isActive"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config"`
}

func (r idpRequest) toDomain() domain.IdentityProvider {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	config := r.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return domain.IdentityProvider{
		Name:        r.Name,
		Description: r.Description,
		IsActive:    isActive,
		Type:        r.Type,
		Config:      config,
	}
}

type idpResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    bool            `json:"isActive"`
	Type        string          `json:"type"`
	Config      json.RawMessage `json:"config"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

func toIdpResponse(idp *domain.IdentityProvider) idpResponse {
	config := idp.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return idpResponse{
		ID:          idp.ID,
		Name:        idp.Name,
		Description: idp.Description,
		IsActive:    idp.IsActive,
		Type:        idp.Type,
		Config:      config,
		CreatedAt:   idp.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   idp.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toIdpResponses(idps []domain.IdentityProvider) []idpResponse {
	resp := make([]idpResponse, 0, len(idps))
	for i := range idps {
		resp = append(resp, toIdpResponse(&idps[i]))
	}
	return resp
}
