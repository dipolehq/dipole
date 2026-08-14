package rest

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

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

	// Sync the display name to the users table so the sidebar reflects immediately.
	email := UserEmailFromContext(r.Context())
	if email != "" {
		if user, err := h.users.GetByEmail(r.Context(), tenantID, email); err == nil && user != nil {
			displayName := strings.TrimSpace(req.FirstName + " " + req.LastName)
			if displayName != "" {
				user.Name = displayName
				user.FirstName = req.FirstName
				user.LastName = req.LastName
				_ = h.users.Update(r.Context(), tenantID, user)
			}
		}
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

	if h.audit != nil {
		h.audit.Created(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Authentication", "api_token", apiToken.ID, name, clientIP(r), userAgent(r))
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

	if h.audit != nil {
		h.audit.Deleted(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Authentication", "api_token", id, "", clientIP(r), userAgent(r))
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

	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Authentication", "password", uuid.Nil, "password-changed", clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully."})
}
