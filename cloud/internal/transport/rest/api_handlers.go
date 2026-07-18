// Package rest — api_handlers.go holds the legacy CRUD handlers for resources
// not yet migrated to the per-resource handler + service pattern.
package rest

import (
	"net/http"


	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type APIHandlers struct {
	accessPolicies   repository.AccessPolicyRepository
	agentControllers repository.AgentControllerRepository
	users            repository.UserRepository
}

func NewAPIHandlers(
	ap repository.AccessPolicyRepository,
	agc repository.AgentControllerRepository,
	u repository.UserRepository,
) *APIHandlers {
	return &APIHandlers{
		accessPolicies:   ap,
		agentControllers: agc,
		users:            u,
	}
}

func (h *APIHandlers) RegisterAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)


	mux.HandleFunc("GET /api/v1/access-policies", h.ListAccessPolicies)
	mux.HandleFunc("POST /api/v1/access-policies", h.CreateAccessPolicy)
	mux.HandleFunc("GET /api/v1/access-policies/{id}", h.GetAccessPolicy)
	mux.HandleFunc("PUT /api/v1/access-policies/{id}", h.UpdateAccessPolicy)
	mux.HandleFunc("DELETE /api/v1/access-policies/{id}", h.DeleteAccessPolicy)

	mux.HandleFunc("GET /api/v1/agent-controllers", h.ListAgentControllers)
	mux.HandleFunc("POST /api/v1/agent-controllers", h.CreateAgentController)
	mux.HandleFunc("GET /api/v1/agent-controllers/{id}", h.GetAgentController)
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


// Access Policies
func (h *APIHandlers) ListAccessPolicies(w http.ResponseWriter, r *http.Request) {
	h.listEntity(w, r, h.accessPolicies)
}
func (h *APIHandlers) CreateAccessPolicy(w http.ResponseWriter, r *http.Request) {
	h.createEntity(w, r, h.accessPolicies)
}
func (h *APIHandlers) GetAccessPolicy(w http.ResponseWriter, r *http.Request) {
	h.getEntity(w, r, h.accessPolicies)
}
func (h *APIHandlers) UpdateAccessPolicy(w http.ResponseWriter, r *http.Request) {
	h.updateEntity(w, r, h.accessPolicies)
}
func (h *APIHandlers) DeleteAccessPolicy(w http.ResponseWriter, r *http.Request) {
	h.deleteEntity(w, r, h.accessPolicies)
}

// Agent Controllers
func (h *APIHandlers) ListAgentControllers(w http.ResponseWriter, r *http.Request) {
	h.listEntity(w, r, h.agentControllers)
}
func (h *APIHandlers) CreateAgentController(w http.ResponseWriter, r *http.Request) {
	h.createEntity(w, r, h.agentControllers)
}
func (h *APIHandlers) GetAgentController(w http.ResponseWriter, r *http.Request) {
	h.getEntity(w, r, h.agentControllers)
}
func (h *APIHandlers) DeleteAgentController(w http.ResponseWriter, r *http.Request) {
	h.deleteEntity(w, r, h.agentControllers)
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
	case repository.AccessPolicyRepository:
		items, total, err := rep.List(r.Context(), tenantID, opts)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, paginatedResponse(items, total, opts))
	case repository.AgentControllerRepository:
		items, total, err := rep.List(r.Context(), tenantID, opts)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, paginatedResponse(items, total, opts))
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
