package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

// --- Identity Provider Handlers ---

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
	if req.Name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	if req.Type == "" {
		respondBadRequest(w, "type is required")
		return
	}
	if req.Type != "saml_2.0" && req.Type != "oidc_1.0" {
		respondBadRequest(w, "type must be saml_2.0 or oidc_1.0")
		return
	}
	idp := req.toDomain()
	if err := h.identityProviders.Create(r.Context(), tenantID, &idp); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.identityProviders.GetByID(r.Context(), tenantID, idp.ID)
	if err != nil || saved == nil {
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
	idp, err := h.identityProviders.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if idp == nil {
		respondError(w, http.StatusNotFound, "not_found", "identity provider not found")
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
	existing, err := h.identityProviders.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "identity provider not found")
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
	idp.Config = preserveSecrets(existing.Config, idp.Config)
	if err := h.identityProviders.Update(r.Context(), tenantID, &idp); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.identityProviders.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
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
	existing, err := h.identityProviders.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "identity provider not found")
		return
	}
	if err := h.identityProviders.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "IdentityProviders", "identity_provider", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

// --- Identity Provider DTOs ---

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

// TODO: Encrypt IdP config at rest (contains clientSecret for OIDC providers).
// For now, config is stored as plaintext JSONB — acceptable for MVP behind auth.
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

// preserveSecrets merges incoming config with the existing config, preserving
// secret fields (clientSecret) when the incoming value is empty. This prevents
// the frontend from accidentally wiping secrets it can't read back (redacted on
// GET responses).
func preserveSecrets(existingRaw, incomingRaw json.RawMessage) json.RawMessage {
	secretFields := []string{"clientSecret"}

	var existing, incoming map[string]json.RawMessage
	if json.Unmarshal(existingRaw, &existing) != nil {
		return incomingRaw
	}
	if json.Unmarshal(incomingRaw, &incoming) != nil {
		return incomingRaw
	}

	for _, field := range secretFields {
		inVal, inOK := incoming[field]
		exVal, exOK := existing[field]
		if !exOK {
			continue
		}
		if !inOK || string(inVal) == `""` || string(inVal) == `null` || string(inVal) == "" {
			incoming[field] = exVal
		}
	}

	merged, err := json.Marshal(incoming)
	if err != nil {
		return incomingRaw
	}
	return merged
}
