/*
auth_handler.go serves the login endpoint for admin users.
*/
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

type AuthHandler struct {
	loginSvc    *service.LoginService
	passwordSvc *service.PasswordResetService
	signupSvc   *service.SignupService
	audit       *AuditEmitter
}

func NewAuthHandler(loginSvc *service.LoginService, passwordSvc *service.PasswordResetService, signupSvc *service.SignupService) *AuthHandler {
	return &AuthHandler{loginSvc: loginSvc, passwordSvc: passwordSvc, signupSvc: signupSvc}
}

func (h *AuthHandler) SetAudit(a *AuditEmitter) { h.audit = a }

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/signup", h.HandleSignup)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", h.HandleForgotPassword)
	mux.HandleFunc("POST /api/v1/auth/reset-password", h.HandleResetPassword)
	mux.HandleFunc("POST /api/v1/auth/mfa-verify", h.HandleMFAVerify)
	mux.HandleFunc("POST /api/v1/auth/switch-tenant", h.HandleSwitchTenant)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.HandleRefresh)
	mux.HandleFunc("GET /api/v1/user/tenants", h.HandleListUserTenants)
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		TenantID string `json:"tenantId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.loginSvc.Login(r.Context(), &service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		TenantID: req.TenantID,
	})
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	if resp.Token != "" && h.audit != nil {
		h.audit.EmitAuditLog(r.Context(), TenantIDFromContext(r.Context()), req.Email, "Authentication", "Login", "user", uuid.Nil, req.Email, clientIP(r), userAgent(r))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.passwordSvc.ForgotPassword(r.Context(), &service.ForgotPasswordRequest{
		Email: req.Email,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	err := h.passwordSvc.ResetPassword(r.Context(), &service.ResetPasswordRequest{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Password has been reset successfully."})
}

func (h *AuthHandler) HandleMFAVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MFAToken string `json:"mfaToken"`
		Code     string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.loginSvc.VerifyMFA(r.Context(), req.MFAToken, req.Code)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleSwitchTenant(w http.ResponseWriter, r *http.Request) {
	email := UserEmailFromContext(r.Context())
	if email == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var req struct {
		TenantID string `json:"tenantId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	targetTenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid tenantId"})
		return
	}

	resp, err := h.loginSvc.SwitchTenant(r.Context(), email, targetTenantID)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleListUserTenants(w http.ResponseWriter, r *http.Request) {
	email := UserEmailFromContext(r.Context())
	if email == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	tenants, err := h.loginSvc.ListTenantsForUser(r.Context(), email)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list tenants"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

func (h *AuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	email := UserEmailFromContext(r.Context())
	tenantID := TenantIDFromContext(r.Context())
	if email == "" || tenantID == uuid.Nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	resp, err := h.loginSvc.RefreshSession(r.Context(), email, tenantID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) HandleSignup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CompanyName string `json:"companyName"`
		Email       string `json:"email"`
		Password    string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.signupSvc.Signup(r.Context(), &service.SignupRequest{
		CompanyName: req.CompanyName,
		Email:       req.Email,
		Password:    req.Password,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if h.audit != nil {
		h.audit.EmitAuditLog(r.Context(), uuid.Nil, req.Email, "Tenant", "Signup", "tenant", uuid.Nil, req.CompanyName, clientIP(r), userAgent(r))
	}
	writeJSON(w, http.StatusCreated, resp)
}
