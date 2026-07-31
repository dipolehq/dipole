/*
auth_handler.go serves the login endpoint for admin users.
*/
package rest

import (
	"encoding/json"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/service"
)

type AuthHandler struct {
	loginSvc    *service.LoginService
	passwordSvc *service.PasswordResetService
}

func NewAuthHandler(loginSvc *service.LoginService, passwordSvc *service.PasswordResetService) *AuthHandler {
	return &AuthHandler{loginSvc: loginSvc, passwordSvc: passwordSvc}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.HandleLogin)
	mux.HandleFunc("POST /api/v1/auth/forgot-password", h.HandleForgotPassword)
	mux.HandleFunc("POST /api/v1/auth/reset-password", h.HandleResetPassword)
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	resp, err := h.loginSvc.Login(r.Context(), &service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
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
