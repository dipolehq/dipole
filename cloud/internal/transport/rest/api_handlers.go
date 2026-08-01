// Package rest — api_handlers.go holds the legacy CRUD handlers for resources
// not yet migrated to the per-resource handler + service pattern.
package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type signOnPolicyStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.SignOnPolicy, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.SignOnPolicy) error
}

type globalPolicyComplianceStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.GlobalPolicyCompliance, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.GlobalPolicyCompliance) error
}

type tenantProfileStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.TenantProfile) error
}

// ControllerCreator abstracts the registration service's controller creation
// so the transport layer doesn't depend on the concrete type.
type ControllerCreator interface {
	CreateController(ctx context.Context, tenantID uuid.UUID, name string) (*domain.AgentController, error)
	RegenerateDeviceCode(ctx context.Context, tenantID, controllerID uuid.UUID) (string, error)
}

type APIHandlers struct {
	agentControllers       repository.AgentControllerRepository
	agentProxies           repository.AgentProxyRepository
	apiTokens              repository.APITokenRepository
	users                  repository.UserRepository
	roles                  repository.RoleRepository
	resourceSets           repository.ResourceSetRepository
	identityProviders      repository.IdentityProviderRepository
	signOnPolicies         signOnPolicyStore
	logStreams             repository.LogStreamRepository
	globalPolicyCompliance globalPolicyComplianceStore
	tenantProfiles         tenantProfileStore
	tenants                repository.TenantRepository
	registrationSvc        ControllerCreator
	planLimiter            *service.PlanLimiter
}

func NewAPIHandlers(
	agc repository.AgentControllerRepository,
	ap repository.AgentProxyRepository,
	at repository.APITokenRepository,
	u repository.UserRepository,
	r repository.RoleRepository,
	rs repository.ResourceSetRepository,
	idp repository.IdentityProviderRepository,
	sop signOnPolicyStore,
	ls repository.LogStreamRepository,
	gpc globalPolicyComplianceStore,
	tp tenantProfileStore,
	regSvc ControllerCreator,
	tenants repository.TenantRepository,
) *APIHandlers {
	return &APIHandlers{
		agentControllers:       agc,
		agentProxies:           ap,
		apiTokens:              at,
		users:                  u,
		roles:                  r,
		resourceSets:           rs,
		identityProviders:      idp,
		signOnPolicies:         sop,
		logStreams:             ls,
		globalPolicyCompliance: gpc,
		tenantProfiles:         tp,
		tenants:                tenants,
		registrationSvc:        regSvc,
	}
}

func (h *APIHandlers) SetPlanLimiter(l *service.PlanLimiter) { h.planLimiter = l }

func (h *APIHandlers) RegisterAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)

	mux.HandleFunc("GET /api/v1/agent-controllers", h.ListAgentControllers)
	mux.HandleFunc("POST /api/v1/agent-controllers", h.CreateAgentController)
	mux.HandleFunc("GET /api/v1/agent-controllers/{id}", h.GetAgentController)
	mux.HandleFunc("PUT /api/v1/agent-controllers/{id}", h.UpdateAgentController)
	mux.HandleFunc("DELETE /api/v1/agent-controllers/{id}", h.DeleteAgentController)
	mux.HandleFunc("GET /api/v1/agent-proxies", h.ListAgentProxies)
	mux.HandleFunc("POST /api/v1/agent-controllers/{id}/device-code", h.RegenerateDeviceCode)

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

	mux.HandleFunc("GET /api/v1/resource-sets", h.ListResourceSets)
	mux.HandleFunc("POST /api/v1/resource-sets", h.CreateResourceSet)
	mux.HandleFunc("GET /api/v1/resource-sets/{id}", h.GetResourceSet)
	mux.HandleFunc("PUT /api/v1/resource-sets/{id}", h.UpdateResourceSet)
	mux.HandleFunc("DELETE /api/v1/resource-sets/{id}", h.DeleteResourceSet)

	mux.HandleFunc("GET /api/v1/identity-providers", h.ListIdentityProviders)
	mux.HandleFunc("POST /api/v1/identity-providers", h.CreateIdentityProvider)
	mux.HandleFunc("GET /api/v1/identity-providers/{id}", h.GetIdentityProvider)
	mux.HandleFunc("PUT /api/v1/identity-providers/{id}", h.UpdateIdentityProvider)
	mux.HandleFunc("DELETE /api/v1/identity-providers/{id}", h.DeleteIdentityProvider)

	mux.HandleFunc("GET /api/v1/sign-on-policy", h.GetSignOnPolicy)
	mux.HandleFunc("PUT /api/v1/sign-on-policy", h.UpdateSignOnPolicy)

	mux.HandleFunc("GET /api/v1/global-policy-compliance", h.GetGlobalPolicyCompliance)
	mux.HandleFunc("PUT /api/v1/global-policy-compliance", h.UpdateGlobalPolicyCompliance)

	mux.HandleFunc("GET /api/v1/profile", h.GetProfile)
	mux.HandleFunc("PUT /api/v1/profile", h.UpdateProfile)
	mux.HandleFunc("POST /api/v1/profile/token", h.GenerateAccessToken)
	mux.HandleFunc("GET /api/v1/profile/tokens", h.ListAccessTokens)
	mux.HandleFunc("DELETE /api/v1/profile/tokens/{id}", h.DeleteAccessToken)
	mux.HandleFunc("POST /api/v1/profile/password", h.UpdatePassword)

	mux.HandleFunc("GET /api/v1/log-streams", h.ListLogStreams)
	mux.HandleFunc("POST /api/v1/log-streams", h.CreateLogStream)
	mux.HandleFunc("GET /api/v1/log-streams/{id}", h.GetLogStream)
	mux.HandleFunc("PUT /api/v1/log-streams/{id}", h.UpdateLogStream)
	mux.HandleFunc("DELETE /api/v1/log-streams/{id}", h.DeleteLogStream)
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
	if h.planLimiter != nil {
		if err := h.planLimiter.Check(r.Context(), tenantID, service.ResourceUsers); err != nil {
			respondBadRequest(w, err.Error())
			return
		}
	}
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

// --- Resource Set Handlers ---

func (h *APIHandlers) ListResourceSets(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.resourceSets.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list resource sets failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toResourceSetResponses(items), total, opts))
}

func (h *APIHandlers) CreateResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req resourceSetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.Name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	rs := req.toDomain()
	if err := h.resourceSets.Create(r.Context(), tenantID, &rs); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.resourceSets.GetByID(r.Context(), tenantID, rs.ID)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusCreated, toResourceSetResponse(&rs))
		return
	}
	respondJSON(w, http.StatusCreated, toResourceSetResponse(saved))
}

func (h *APIHandlers) GetResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	rs, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if rs == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	respondJSON(w, http.StatusOK, toResourceSetResponse(rs))
}

func (h *APIHandlers) UpdateResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	var req resourceSetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	rs := req.toDomain()
	rs.ID = id
	rs.TenantID = tenantID
	if err := h.resourceSets.Update(r.Context(), tenantID, &rs); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toResourceSetResponse(&rs))
		return
	}
	respondJSON(w, http.StatusOK, toResourceSetResponse(saved))
}

func (h *APIHandlers) DeleteResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	if err := h.resourceSets.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Resource Set DTOs ---

type resourceSetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r resourceSetRequest) toDomain() domain.ResourceSet {
	return domain.ResourceSet{
		Name:        r.Name,
		Description: r.Description,
	}
}

type resourceSetResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
}

func toResourceSetResponse(rs *domain.ResourceSet) resourceSetResponse {
	return resourceSetResponse{
		ID:          rs.ID,
		Name:        rs.Name,
		Description: rs.Description,
		CreatedAt:   rs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   rs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toResourceSetResponses(sets []domain.ResourceSet) []resourceSetResponse {
	resp := make([]resourceSetResponse, 0, len(sets))
	for i := range sets {
		resp = append(resp, toResourceSetResponse(&sets[i]))
	}
	return resp
}

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
		respondJSON(w, http.StatusCreated, toIdpResponse(&idp))
		return
	}
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
	if err := h.identityProviders.Update(r.Context(), tenantID, &idp); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.identityProviders.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toIdpResponse(&idp))
		return
	}
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

// --- Sign-On Policy Handlers ---

func (h *APIHandlers) GetSignOnPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	p, err := h.signOnPolicies.Get(r.Context(), tenantID)
	if err != nil {
		slog.Error("get sign-on policy failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"requireSso": p.RequireSSO,
		"requireMfa": p.RequireMFA,
	})
}

func (h *APIHandlers) UpdateSignOnPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req struct {
		RequireSSO bool `json:"requireSso"`
		RequireMFA bool `json:"requireMfa"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p := &domain.SignOnPolicy{
		TenantID:   tenantID,
		RequireSSO: req.RequireSSO,
		RequireMFA: req.RequireMFA,
	}
	if err := h.signOnPolicies.Upsert(r.Context(), tenantID, p); err != nil {
		slog.Error("update sign-on policy failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"requireSso": req.RequireSSO,
		"requireMfa": req.RequireMFA,
	})
}

// --- Log Stream Handlers ---

func (h *APIHandlers) ListLogStreams(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.logStreams.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list log streams failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, NewListResponse(toLogStreamResponses(items), total, opts))
}

func (h *APIHandlers) CreateLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if h.planLimiter != nil {
		if err := h.planLimiter.Check(r.Context(), tenantID, service.ResourceLogStreams); err != nil {
			respondBadRequest(w, err.Error())
			return
		}
	}
	var req logStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.Name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	if req.EventType == "" {
		respondBadRequest(w, "eventType is required")
		return
	}
	if req.DestinationType == "" {
		respondBadRequest(w, "destinationType is required")
		return
	}
	validEventTypes := map[string]bool{"access_authorization": true, "audit_logs": true, "workload_events": true}
	if !validEventTypes[req.EventType] {
		respondBadRequest(w, "eventType must be access_authorization, audit_logs, or workload_events")
		return
	}
	validDestTypes := map[string]bool{"aws_s3": true, "gcs_bucket": true, "crowdstrike_siem": true, "splunk_siem": true}
	if !validDestTypes[req.DestinationType] {
		respondBadRequest(w, "destinationType must be aws_s3, gcs_bucket, crowdstrike_siem, or splunk_siem")
		return
	}
	ls := req.toDomain()
	if err := h.logStreams.Create(r.Context(), tenantID, &ls); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.logStreams.GetByID(r.Context(), tenantID, ls.ID)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusCreated, toLogStreamResponse(&ls))
		return
	}
	respondJSON(w, http.StatusCreated, toLogStreamResponse(saved))
}

func (h *APIHandlers) GetLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	ls, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if ls == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	respondJSON(w, http.StatusOK, toLogStreamResponse(ls))
}

func (h *APIHandlers) UpdateLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	var req logStreamRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	ls := req.toDomain()
	ls.ID = id
	ls.TenantID = tenantID
	if err := h.logStreams.Update(r.Context(), tenantID, &ls); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		respondJSON(w, http.StatusOK, toLogStreamResponse(&ls))
		return
	}
	respondJSON(w, http.StatusOK, toLogStreamResponse(saved))
}

func (h *APIHandlers) DeleteLogStream(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.logStreams.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "log stream not found")
		return
	}
	if err := h.logStreams.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Log Stream DTOs ---

type logStreamRequest struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	IsActive        *bool           `json:"isActive"`
	EventType       string          `json:"eventType"`
	DestinationType string          `json:"destinationType"`
	Config          json.RawMessage `json:"config"`
}

func (r logStreamRequest) toDomain() domain.LogStream {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}
	config := r.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return domain.LogStream{
		Name:            r.Name,
		Description:     r.Description,
		IsActive:        isActive,
		EventType:       r.EventType,
		DestinationType: r.DestinationType,
		Config:          config,
	}
}

type logStreamResponse struct {
	ID              uuid.UUID       `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	IsActive        bool            `json:"isActive"`
	EventType       string          `json:"eventType"`
	DestinationType string          `json:"destinationType"`
	Config          json.RawMessage `json:"config"`
	CreatedAt       string          `json:"createdAt"`
	UpdatedAt       string          `json:"updatedAt"`
}

// TODO: Encrypt Log Stream config at rest (contains apiKey/authToken for SIEM destinations).
// For now, config is stored as plaintext JSONB — acceptable for MVP behind auth.
func toLogStreamResponse(ls *domain.LogStream) logStreamResponse {
	config := ls.Config
	if config == nil {
		config = json.RawMessage(`{}`)
	}
	return logStreamResponse{
		ID:              ls.ID,
		Name:            ls.Name,
		Description:     ls.Description,
		IsActive:        ls.IsActive,
		EventType:       ls.EventType,
		DestinationType: ls.DestinationType,
		Config:          config,
		CreatedAt:       ls.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       ls.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toLogStreamResponses(streams []domain.LogStream) []logStreamResponse {
	resp := make([]logStreamResponse, 0, len(streams))
	for i := range streams {
		resp = append(resp, toLogStreamResponse(&streams[i]))
	}
	return resp
}

// --- Global Policy Compliance Handlers ---

func (h *APIHandlers) GetGlobalPolicyCompliance(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	p, err := h.globalPolicyCompliance.Get(r.Context(), tenantID)
	if err != nil {
		slog.Error("get global policy compliance failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"apTrustProvider":            p.APTrustProvider,
		"apAccessCondition":          p.APAccessCondition,
		"acTrustProvider":            p.ACTrustProvider,
		"acTlsHostname":              p.ACTLSHostname,
		"accessTokenLifetimeSeconds": p.AccessTokenLifetimeSeconds,
	})
}

func (h *APIHandlers) UpdateGlobalPolicyCompliance(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req struct {
		APTrustProvider            string `json:"apTrustProvider"`
		APAccessCondition          string `json:"apAccessCondition"`
		ACTrustProvider            string `json:"acTrustProvider"`
		ACTLSHostname              string `json:"acTlsHostname"`
		AccessTokenLifetimeSeconds int    `json:"accessTokenLifetimeSeconds"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p := &domain.GlobalPolicyCompliance{
		TenantID:                   tenantID,
		APTrustProvider:            req.APTrustProvider,
		APAccessCondition:          req.APAccessCondition,
		ACTrustProvider:            req.ACTrustProvider,
		ACTLSHostname:              req.ACTLSHostname,
		AccessTokenLifetimeSeconds: req.AccessTokenLifetimeSeconds,
	}
	if err := h.globalPolicyCompliance.Upsert(r.Context(), tenantID, p); err != nil {
		slog.Error("update global policy compliance failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"apTrustProvider":            req.APTrustProvider,
		"apAccessCondition":          req.APAccessCondition,
		"acTrustProvider":            req.ACTrustProvider,
		"acTlsHostname":              req.ACTLSHostname,
		"accessTokenLifetimeSeconds": req.AccessTokenLifetimeSeconds,
	})
}

// --- Profile Handlers ---

func (h *APIHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	p, err := h.tenantProfiles.Get(r.Context(), tenantID)
	if err != nil || p == nil {
		// No profile yet — return empty defaults
		respondJSON(w, http.StatusOK, map[string]any{
			"firstName": "",
			"lastName":  "",
			"email":     "",
			"phone":     "",
		})
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"firstName": p.FirstName,
		"lastName":  p.LastName,
		"email":     p.Email,
		"phone":     p.Phone,
	})
}

func (h *APIHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
		Phone     string `json:"phone"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p := &domain.TenantProfile{
		TenantID:  tenantID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
	}
	if err := h.tenantProfiles.Upsert(r.Context(), tenantID, p); err != nil {
		slog.Error("update profile failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"firstName": req.FirstName,
		"lastName":  req.LastName,
		"email":     req.Email,
		"phone":     req.Phone,
	})
}

func (h *APIHandlers) GenerateAccessToken(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	var req struct {
		Name          string  `json:"name"`
		ResourceSetID *string `json:"resourceSetId"`
	}
	// Body is optional for backward compatibility (existing UI sends empty body).
	_ = decodeJSON(r, &req)

	token, err := service.GenerateAPIToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to generate token")
		return
	}

	hash := service.HashAPIToken(token)

	var rsID *uuid.UUID
	if req.ResourceSetID != nil && *req.ResourceSetID != "" {
		parsed, err := uuid.Parse(*req.ResourceSetID)
		if err != nil {
			respondBadRequest(w, "invalid resourceSetId")
			return
		}
		rsID = &parsed
	}

	name := req.Name
	if name == "" {
		name = "API Token"
	}

	apiToken := &domain.APIToken{
		Name:          name,
		TokenHash:     hash,
		ResourceSetID: rsID,
	}
	if err := h.apiTokens.Create(r.Context(), tenantID, apiToken); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to store token")
		return
	}

	// Also update the legacy tenant-level token for backward compatibility.
	tenant, err := h.tenants.GetByID(r.Context(), tenantID)
	if err == nil && tenant != nil {
		tenant.APITokenHash = hash
		_ = h.tenants.Update(r.Context(), tenant)
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token":         token,
		"id":            apiToken.ID.String(),
		"name":          name,
		"resourceSetId": rsID,
	})
}

func (h *APIHandlers) ListAccessTokens(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	tokens, err := h.apiTokens.List(r.Context(), tenantID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to list tokens")
		return
	}

	type tokenResponse struct {
		ID            string  `json:"id"`
		Name          string  `json:"name"`
		ResourceSetID *string `json:"resourceSetId"`
		CreatedAt     string  `json:"createdAt"`
	}

	resp := make([]tokenResponse, 0, len(tokens))
	for _, t := range tokens {
		tr := tokenResponse{
			ID:        t.ID.String(),
			Name:      t.Name,
			CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if t.ResourceSetID != nil {
			s := t.ResourceSetID.String()
			tr.ResourceSetID = &s
		}
		resp = append(resp, tr)
	}

	respondJSON(w, http.StatusOK, map[string]any{"tokens": resp})
}

func (h *APIHandlers) DeleteAccessToken(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid token id")
		return
	}

	if err := h.apiTokens.Delete(r.Context(), tenantID, id); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to delete token")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *APIHandlers) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, "invalid request body")
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		respondBadRequest(w, "currentPassword and newPassword are required")
		return
	}
	if len(req.NewPassword) < 6 {
		respondBadRequest(w, "new password must be at least 6 characters")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	// Verify current password.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		respondBadRequest(w, "current password is incorrect")
		return
	}

	// Hash and update.
	newHash, err := service.HashPassword(req.NewPassword)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}

	user.PasswordHash = newHash
	if err := h.users.Update(r.Context(), tenantID, user); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to update password")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully."})
}
