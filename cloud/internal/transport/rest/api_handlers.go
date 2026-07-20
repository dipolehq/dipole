// Package rest — api_handlers.go holds the legacy CRUD handlers for resources
// not yet migrated to the per-resource handler + service pattern.
package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/google/uuid"
)

type APIHandlers struct {
	agentControllers repository.AgentControllerRepository
	users            repository.UserRepository
	roles            repository.RoleRepository
}

func NewAPIHandlers(
	agc repository.AgentControllerRepository,
	u repository.UserRepository,
	r repository.RoleRepository,
) *APIHandlers {
	return &APIHandlers{
		agentControllers: agc,
		users:            u,
		roles:            r,
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

	mux.HandleFunc("GET /api/v1/roles", h.ListRoles)
	mux.HandleFunc("POST /api/v1/roles", h.CreateRole)
	mux.HandleFunc("GET /api/v1/roles/{id}", h.GetRole)
	mux.HandleFunc("PUT /api/v1/roles/{id}", h.UpdateRole)
	mux.HandleFunc("DELETE /api/v1/roles/{id}", h.DeleteRole)
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
		slog.Error("list failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
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
func (h *APIHandlers) ListUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.users.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toUserResponses(items), total, opts))
}

func (h *APIHandlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req userRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.Email == "" {
		respondBadRequest(w, "email is required")
		return
	}
	if req.FirstName == "" {
		respondBadRequest(w, "firstName is required")
		return
	}
	u := req.toDomain()
	if err := h.users.Create(r.Context(), tenantID, &u); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.users.GetByID(r.Context(), tenantID, u.ID)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusCreated, toUserResponse(&u))
		return
	}
	respondJSON(w, http.StatusCreated, toUserResponse(saved))
}

func (h *APIHandlers) GetUser(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	u, err := h.users.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if u == nil {
		respondError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	respondJSON(w, http.StatusOK, toUserResponse(u))
}

func (h *APIHandlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.users.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	var req userRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	u := req.toDomain()
	u.ID = id
	u.TenantID = tenantID
	if err := h.users.Update(r.Context(), tenantID, &u); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.users.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toUserResponse(&u))
		return
	}
	respondJSON(w, http.StatusOK, toUserResponse(saved))
}

func (h *APIHandlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.users.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}
	if err := h.users.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Roles
func (h *APIHandlers) ListRoles(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.roles.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toRoleResponses(items), total, opts))
}

func (h *APIHandlers) CreateRole(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req roleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	role := req.toDomain()
	if err := h.roles.Create(r.Context(), tenantID, &role); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.roles.GetByID(r.Context(), tenantID, role.ID)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusCreated, toRoleResponse(&role))
		return
	}
	respondJSON(w, http.StatusCreated, toRoleResponse(saved))
}

func (h *APIHandlers) GetRole(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	role, err := h.roles.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if role == nil {
		respondError(w, http.StatusNotFound, "not_found", "role not found")
		return
	}
	respondJSON(w, http.StatusOK, toRoleResponse(role))
}

func (h *APIHandlers) UpdateRole(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.roles.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "role not found")
		return
	}
	var req roleRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	role := req.toDomain()
	role.ID = id
	role.TenantID = tenantID
	if err := h.roles.Update(r.Context(), tenantID, &role); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.roles.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toRoleResponse(&role))
		return
	}
	respondJSON(w, http.StatusOK, toRoleResponse(saved))
}

func (h *APIHandlers) DeleteRole(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.roles.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "role not found")
		return
	}
	if err := h.roles.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// --- User DTOs ---

type userRequest struct {
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	IsActive  *bool      `json:"isActive"`
	RoleID    *uuid.UUID `json:"roleId"`
}

func (r userRequest) toDomain() domain.User {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	name := r.FirstName
	if r.LastName != "" {
		name = r.FirstName + " " + r.LastName
	}
	return domain.User{
		Email:     r.Email,
		Name:      name,
		FirstName: r.FirstName,
		LastName:  r.LastName,
		Phone:     r.Phone,
		IsActive:  isActive,
		RoleID:    r.RoleID,
	}
}

type userResponse struct {
	ID        uuid.UUID  `json:"id"`
	FirstName string     `json:"firstName"`
	LastName  string     `json:"lastName"`
	Email     string     `json:"email"`
	Phone     string     `json:"phone"`
	IsActive  bool       `json:"isActive"`
	RoleID    *uuid.UUID `json:"roleId,omitempty"`
	CreatedAt string     `json:"createdAt"`
	UpdatedAt string     `json:"updatedAt"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		FirstName: u.FirstName,
		LastName:  u.LastName,
		Email:     u.Email,
		Phone:     u.Phone,
		IsActive:  u.IsActive,
		RoleID:    u.RoleID,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toUserResponses(users []domain.User) []userResponse {
	resp := make([]userResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}
	return resp
}

// --- Role DTOs ---

type roleRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    *bool           `json:"isActive"`
	Permissions json.RawMessage `json:"permissions"`
}

func (r roleRequest) toDomain() domain.Role {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	perms := r.Permissions
	if perms == nil {
		perms = json.RawMessage(`{}`)
	}
	return domain.Role{
		Name:        r.Name,
		Description: r.Description,
		IsActive:    isActive,
		Permissions: perms,
	}
}

type roleResponse struct {
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	IsActive    bool            `json:"isActive"`
	Permissions json.RawMessage `json:"permissions"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

func toRoleResponse(r *domain.Role) roleResponse {
	perms := r.Permissions
	if perms == nil {
		perms = json.RawMessage(`{}`)
	}
	return roleResponse{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		IsActive:    r.IsActive,
		Permissions: perms,
		CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toRoleResponses(roles []domain.Role) []roleResponse {
	resp := make([]roleResponse, 0, len(roles))
	for i := range roles {
		resp = append(resp, toRoleResponse(&roles[i]))
	}
	return resp
}
