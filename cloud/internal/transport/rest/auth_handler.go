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
	loginSvc *service.LoginService
}

func NewAuthHandler(loginSvc *service.LoginService) *AuthHandler {
	return &AuthHandler{loginSvc: loginSvc}
}

func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/auth/login", h.HandleLogin)
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
