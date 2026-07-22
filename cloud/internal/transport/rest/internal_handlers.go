package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/service/registration"
)

// InternalHandlers serves unauthenticated internal endpoints used by
// Edge Agent Controllers for registration, token exchange, and health reporting.
type InternalHandlers struct {
	registrationSvc *registration.RegistrationService
}

func NewInternalHandlers(registrationSvc *registration.RegistrationService) *InternalHandlers {
	return &InternalHandlers{registrationSvc: registrationSvc}
}

func (h *InternalHandlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /internal/controllers/register", h.handleControllerRegister)
	mux.HandleFunc("POST /internal/controllers/token", h.handleControllerToken)
	mux.HandleFunc("POST /internal/controllers/health", h.handleControllerHealth)
}

func (h *InternalHandlers) handleControllerRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID   string `json:"tenant_id"`
		DeviceCode string `json:"device_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid tenant_id")
		return
	}
	if req.DeviceCode == "" {
		respondError(w, http.StatusBadRequest, "bad_request", "device_code is required")
		return
	}

	controller, err := h.registrationSvc.RegisterController(r.Context(), tenantID, req.DeviceCode)
	if err != nil {
		respondError(w, http.StatusBadRequest, "registration_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"controller_id": controller.ID.String(),
	})
}

func (h *InternalHandlers) handleControllerToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		ControllerID string `json:"controller_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid tenant_id")
		return
	}
	controllerID, err := uuid.Parse(req.ControllerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid controller_id")
		return
	}

	token, err := h.registrationSvc.IssueProxyToken(r.Context(), tenantID, controllerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "token_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

func (h *InternalHandlers) handleControllerHealth(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantID     string `json:"tenant_id"`
		ControllerID string `json:"controller_id"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid tenant_id")
		return
	}
	controllerID, err := uuid.Parse(req.ControllerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid controller_id")
		return
	}

	if err := h.registrationSvc.UpdateHealth(r.Context(), tenantID, controllerID); err != nil {
		respondError(w, http.StatusInternalServerError, "health_update_failed", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{
		"acknowledged": true,
	})
}
