package rest

import (
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

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
		emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Users", "user", u.ID, u.Email, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusCreated, toUserResponse(&u))
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Users", "user", saved.ID, saved.Email, clientIP(r), userAgent(r))
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
		emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Users", "user", u.ID, u.Email, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusOK, toUserResponse(&u))
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Users", "user", saved.ID, saved.Email, clientIP(r), userAgent(r))
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
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "Users", "user", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
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
