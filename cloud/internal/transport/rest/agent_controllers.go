package rest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

// Agent Controllers
func (h *APIHandlers) ListAgentControllers(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.agentControllers.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toAgentControllerResponses(items), total, opts))
}

func (h *APIHandlers) ListAgentProxies(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.agentProxies.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list agent proxies failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	type proxyResponse struct {
		ID                string  `json:"id"`
		Status            string  `json:"status"`
		AgentControllerID *string `json:"agentControllerId"`
		LastHealthAt      *string `json:"lastHealthAt"`
		ResourceSetID     *string `json:"resourceSetId"`
		RegisteredAt      *string `json:"registeredAt"`
	}
	resp := make([]proxyResponse, 0, len(items))
	for _, p := range items {
		pr := proxyResponse{
			ID:     p.ID.String(),
			Status: p.Status,
		}
		if p.AgentControllerID != nil {
			s := p.AgentControllerID.String()
			pr.AgentControllerID = &s
		}
		if p.LastHealthAt != nil {
			s := p.LastHealthAt.Format("2006-01-02T15:04:05Z")
			pr.LastHealthAt = &s
		}
		if p.ResourceSetID != nil {
			s := p.ResourceSetID.String()
			pr.ResourceSetID = &s
		}
		if p.RegisteredAt != nil {
			s := p.RegisteredAt.Format("2006-01-02T15:04:05Z")
			pr.RegisteredAt = &s
		}
		resp = append(resp, pr)
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *APIHandlers) CreateAgentController(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if h.planLimiter != nil {
		if err := h.planLimiter.Check(r.Context(), tenantID, service.ResourceAgentControllers); err != nil {
			respondBadRequest(w, err.Error())
			return
		}
	}
	var req agentControllerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	// Use RegistrationService to create with device code + expiry
	ac, err := h.registrationSvc.CreateController(r.Context(), tenantID, req.Name)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	// Apply additional fields from request that CreateController doesn't handle
	needsUpdate := false
	if req.Description != "" {
		ac.Description = req.Description
		needsUpdate = true
	}
	if req.TrustProviderID != nil {
		if err := h.validateTrustProviderBelongsToTenant(r.Context(), tenantID, *req.TrustProviderID); err != nil {
			respondDomainError(w, err)
			return
		}
		ac.TrustProviderID = req.TrustProviderID
		needsUpdate = true
	}
	if req.TLSHostname != "" {
		ac.TLSHostname = req.TLSHostname
		needsUpdate = true
	}
	if req.ResourceSetID != nil {
		ac.ResourceSetID = req.ResourceSetID
		needsUpdate = true
	}
	if len(req.Tags) > 0 {
		tags := make([]domain.Tag, 0, len(req.Tags))
		for _, t := range req.Tags {
			tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
		}
		ac.Tags = tags
		needsUpdate = true
	}
	if req.IsActive != nil && !*req.IsActive {
		ac.IsActive = false
		needsUpdate = true
	}

	// If additional fields were set, update the record
	if needsUpdate {
		if err := h.agentControllers.Update(r.Context(), tenantID, ac); err != nil {
			respondDomainError(w, err)
			return
		}
	}

	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AgentControllers", "agent_controller", ac.ID, ac.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toAgentControllerResponse(ac))
}

func (h *APIHandlers) GetAgentController(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	ac, err := h.agentControllers.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if ac == nil {
		respondError(w, http.StatusNotFound, "not_found", "agent controller not found")
		return
	}
	respondJSON(w, http.StatusOK, toAgentControllerResponse(ac))
}

func (h *APIHandlers) UpdateAgentController(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.agentControllers.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "agent controller not found")
		return
	}
	var req agentControllerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.TrustProviderID != nil {
		if err := h.validateTrustProviderBelongsToTenant(r.Context(), tenantID, *req.TrustProviderID); err != nil {
			respondDomainError(w, err)
			return
		}
	}
	ac := req.toDomain()
	ac.ID = id
	ac.TenantID = tenantID
	if ac.ResourceSetID == nil {
		ac.ResourceSetID = existing.ResourceSetID
	}
	if err := h.agentControllers.Update(r.Context(), tenantID, &ac); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.agentControllers.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AgentControllers", "agent_controller", ac.ID, ac.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusOK, toAgentControllerResponse(&ac))
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AgentControllers", "agent_controller", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toAgentControllerResponse(saved))
}

func (h *APIHandlers) DeleteAgentController(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.agentControllers.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "agent controller not found")
		return
	}
	if err := h.agentControllers.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AgentControllers", "agent_controller", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

// validateTrustProviderBelongsToTenant confirms trustProviderID belongs to
// tenantID before an Agent Controller is allowed to reference it. Without
// this check, a tenant could attach another tenant's Trust Provider ID
// (guessed, leaked, or otherwise obtained) to their own controller, which
// would then be used for real attestation via the edge trust validator.
// TrustProviderRepository.GetByID is already tenant-scoped, so a
// cross-tenant ID resolves to nil exactly like a nonexistent one.
func (h *APIHandlers) validateTrustProviderBelongsToTenant(ctx context.Context, tenantID, trustProviderID uuid.UUID) error {
	tp, err := h.trustProviders.GetByID(ctx, tenantID, trustProviderID)
	if err != nil {
		return fmt.Errorf("resolve trust provider: %w", err)
	}
	if tp == nil {
		return domain.NewValidationError("trustProviderId", "trust provider not found")
	}
	return nil
}

func (h *APIHandlers) RegenerateDeviceCode(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	code, err := h.registrationSvc.RegenerateDeviceCode(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"deviceCode": code})
}

// --- Agent Controller DTOs ---

type agentControllerRequest struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	IsActive        *bool      `json:"isActive"`
	TrustProviderID *uuid.UUID `json:"trustProviderId"`
	TLSHostname     string     `json:"tlsHostname"`
	Tags            []tagDTO   `json:"tags"`
	ResourceSetID   *uuid.UUID `json:"resourceSetId"`
}

func (r agentControllerRequest) toDomain() domain.AgentController {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	status := "unregistered"
	if isActive {
		status = "healthy"
	}
	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}
	return domain.AgentController{
		Name:            r.Name,
		Description:     r.Description,
		IsActive:        isActive,
		Status:          status,
		TrustProviderID: r.TrustProviderID,
		TLSHostname:     r.TLSHostname,
		Tags:            tags,
		ResourceSetID:   r.ResourceSetID,
	}
}

type agentControllerResponse struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	IsActive        bool       `json:"isActive"`
	Status          string     `json:"status"`
	DeviceCode      string     `json:"deviceCode,omitempty"`
	TrustProviderID *uuid.UUID `json:"trustProviderId,omitempty"`
	TLSHostname     string     `json:"tlsHostname,omitempty"`
	Tags            []tagDTO   `json:"tags"`
	ResourceSetID   *uuid.UUID `json:"resourceSetId,omitempty"`
	CreatedAt       string     `json:"createdAt"`
	UpdatedAt       string     `json:"updatedAt"`
}

func toAgentControllerResponse(ac *domain.AgentController) agentControllerResponse {
	tags := make([]tagDTO, 0, len(ac.Tags))
	for _, t := range ac.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}
	return agentControllerResponse{
		ID:              ac.ID,
		Name:            ac.Name,
		Description:     ac.Description,
		IsActive:        ac.IsActive,
		Status:          ac.Status,
		DeviceCode:      ac.DeviceCode,
		TrustProviderID: ac.TrustProviderID,
		TLSHostname:     ac.TLSHostname,
		Tags:            tags,
		ResourceSetID:   ac.ResourceSetID,
		CreatedAt:       ac.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       ac.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toAgentControllerResponses(acs []domain.AgentController) []agentControllerResponse {
	resp := make([]agentControllerResponse, 0, len(acs))
	for i := range acs {
		resp = append(resp, toAgentControllerResponse(&acs[i]))
	}
	return resp
}
