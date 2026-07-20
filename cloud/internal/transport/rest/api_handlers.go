// Package rest — api_handlers.go holds the legacy CRUD handlers for resources
// not yet migrated to the per-resource handler + service pattern.
package rest

import (
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/google/uuid"
)

type APIHandlers struct {
	agentControllers repository.AgentControllerRepository
	users            repository.UserRepository
}

func NewAPIHandlers(
	agc repository.AgentControllerRepository,
	u repository.UserRepository,
) *APIHandlers {
	return &APIHandlers{
		agentControllers: agc,
		users:            u,
	}
}

func (h *APIHandlers) RegisterAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)

	mux.HandleFunc("GET /api/v1/agent-controllers", h.ListAgentControllers)
	mux.HandleFunc("POST /api/v1/agent-controllers", h.CreateAgentController)
	mux.HandleFunc("GET /api/v1/agent-controllers/{id}", h.GetAgentController)
	mux.HandleFunc("PUT /api/v1/agent-controllers/{id}", h.UpdateAgentController)
	mux.HandleFunc("DELETE /api/v1/agent-controllers/{id}", h.DeleteAgentController)

	mux.HandleFunc("GET /api/v1/users", h.ListUsers)
	mux.HandleFunc("POST /api/v1/users", h.CreateUser)
	mux.HandleFunc("GET /api/v1/users/{id}", h.GetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", h.DeleteUser)
}

func (h *APIHandlers) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "Healthy"})
}

// Agent Controllers
func (h *APIHandlers) ListAgentControllers(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.agentControllers.List(r.Context(), tenantID, opts)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toAgentControllerResponses(items), total, opts))
}

func (h *APIHandlers) CreateAgentController(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req agentControllerRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ac := req.toDomain()
	if err := h.agentControllers.Create(r.Context(), tenantID, &ac); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.agentControllers.GetByID(r.Context(), tenantID, ac.ID)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusCreated, toAgentControllerResponse(&ac))
		return
	}
	respondJSON(w, http.StatusCreated, toAgentControllerResponse(saved))
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
	ac := req.toDomain()
	ac.ID = id
	ac.TenantID = tenantID
	if err := h.agentControllers.Update(r.Context(), tenantID, &ac); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.agentControllers.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toAgentControllerResponse(&ac))
		return
	}
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
	w.WriteHeader(http.StatusNoContent)
}

// Users
func (h *APIHandlers) ListUsers(w http.ResponseWriter, r *http.Request) { h.listEntity(w, r, h.users) }
func (h *APIHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	h.createEntity(w, r, h.users)
}
func (h *APIHandlers) GetUser(w http.ResponseWriter, r *http.Request) { h.getEntity(w, r, h.users) }
func (h *APIHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	h.updateEntity(w, r, h.users)
}
func (h *APIHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.deleteEntity(w, r, h.users)
}

func (h *APIHandlers) listEntity(w http.ResponseWriter, r *http.Request, repo any) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)

	switch rep := repo.(type) {
	case repository.UserRepository:
		items, total, err := rep.List(r.Context(), tenantID, opts)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, paginatedResponse(items, total, opts))
	default:
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "unsupported entity type"})
	}
}

func (h *APIHandlers) createEntity(w http.ResponseWriter, _ *http.Request, _ any) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "create not implemented for this entity"})
}

func (h *APIHandlers) getEntity(w http.ResponseWriter, _ *http.Request, _ any) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "get not implemented for this entity"})
}

func (h *APIHandlers) updateEntity(w http.ResponseWriter, _ *http.Request, _ any) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "update not implemented for this entity"})
}

func (h *APIHandlers) deleteEntity(w http.ResponseWriter, _ *http.Request, _ any) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "delete not implemented for this entity"})
}

func paginatedResponse(items any, total int, opts domain.ListOptions) map[string]any {
	return map[string]any{
		"items":   items,
		"total":   total,
		"page":    opts.Page,
		"perPage": opts.PerPage,
	}
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
