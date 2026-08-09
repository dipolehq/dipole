package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

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
		if h.audit != nil {
			h.audit.Created(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Roles", "role", role.ID, role.Name, clientIP(r), userAgent(r))
		}
		respondJSON(w, http.StatusCreated, toRoleResponse(&role))
		return
	}
	if h.audit != nil {
		h.audit.Created(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Roles", "role", saved.ID, saved.Name, clientIP(r), userAgent(r))
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
		if h.audit != nil {
			h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Roles", "role", role.ID, role.Name, clientIP(r), userAgent(r))
		}
		respondJSON(w, http.StatusOK, toRoleResponse(&role))
		return
	}
	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Roles", "role", saved.ID, saved.Name, clientIP(r), userAgent(r))
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
	if h.audit != nil {
		h.audit.Deleted(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Roles", "role", id, "", clientIP(r), userAgent(r))
	}
	w.WriteHeader(http.StatusNoContent)
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
