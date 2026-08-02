/*
mfa_handler.go serves the MFA (multi-factor authentication) profile endpoints:
  - GET    /api/v1/profile/mfa/status       — current MFA state
  - POST   /api/v1/profile/mfa/setup        — generate TOTP secret + URI
  - POST   /api/v1/profile/mfa/verify       — verify code and enable MFA
  - POST   /api/v1/profile/mfa/backup-codes — generate backup codes
  - DELETE /api/v1/profile/mfa              — reset (remove) MFA
*/
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type MFAHandler struct {
	mfaSvc *service.MFAService
	users  repository.UserRepository
	audit  *AuditEmitter
}

func NewMFAHandler(mfaSvc *service.MFAService, users repository.UserRepository) *MFAHandler {
	return &MFAHandler{mfaSvc: mfaSvc, users: users}
}

func (h *MFAHandler) SetAudit(a *AuditEmitter) { h.audit = a }

func (h *MFAHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/profile/mfa/status", h.GetStatus)
	mux.HandleFunc("POST /api/v1/profile/mfa/setup", h.Setup)
	mux.HandleFunc("POST /api/v1/profile/mfa/verify", h.Verify)
	mux.HandleFunc("POST /api/v1/profile/mfa/backup-codes", h.GenerateBackupCodes)
	mux.HandleFunc("DELETE /api/v1/profile/mfa", h.Reset)
}

func (h *MFAHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	status, err := h.mfaSvc.GetStatus(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to get MFA status")
		return
	}

	respondJSON(w, http.StatusOK, status)
}

func (h *MFAHandler) Setup(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	resp, err := h.mfaSvc.Setup(r.Context(), user.ID, email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to set up MFA")
		return
	}

	respondJSON(w, http.StatusOK, resp)
}

func (h *MFAHandler) Verify(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondBadRequest(w, "invalid request body")
		return
	}
	if req.Code == "" {
		respondBadRequest(w, "code is required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	if err := h.mfaSvc.Verify(r.Context(), user.ID, req.Code); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	if h.audit != nil {
		h.audit.EmitAuditLog(r.Context(), tenantID, email, "Authentication", "MFA Enabled", "user", user.ID, email, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "MFA enabled successfully."})
}

func (h *MFAHandler) GenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	codes, err := h.mfaSvc.GenerateBackupCodes(r.Context(), user.ID)
	if err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"codes": codes})
}

func (h *MFAHandler) Reset(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	email := UserEmailFromContext(r.Context())

	user, err := h.users.GetByEmail(r.Context(), tenantID, email)
	if err != nil || user == nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "user not found")
		return
	}

	if err := h.mfaSvc.Reset(r.Context(), user.ID); err != nil {
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to reset MFA")
		return
	}

	if h.audit != nil {
		h.audit.EmitAuditLog(r.Context(), tenantID, email, "Authentication", "MFA Disabled", "user", user.ID, email, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, map[string]string{"message": "MFA has been reset."})
}
