// internal_handlers.go serves internal endpoints used by Edge Agent Controllers for registration, token exchange, and health reporting, authenticated via a per-controller HMAC signature (except /register, which uses a device code instead).
package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/registration"
)

const controllerSignatureHeader = "X-Dipole-Controller-Signature"

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

	controller, sharedSecret, err := h.registrationSvc.RegisterController(r.Context(), tenantID, req.DeviceCode)
	if err != nil {
		respondRegistrationError(w, "registration_failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"controller_id": controller.ID.String(),
		"shared_secret": sharedSecret,
	})
}

func (h *InternalHandlers) handleControllerToken(w http.ResponseWriter, r *http.Request) {
	body, tenantID, controllerID, ok := h.readAndVerifySignedRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		TenantID     string `json:"tenant_id"`
		ControllerID string `json:"controller_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	token, err := h.registrationSvc.IssueProxyToken(r.Context(), tenantID, controllerID)
	if err != nil {
		respondRegistrationError(w, "token_failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

func (h *InternalHandlers) handleControllerHealth(w http.ResponseWriter, r *http.Request) {
	body, tenantID, controllerID, ok := h.readAndVerifySignedRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		TenantID     string `json:"tenant_id"`
		ControllerID string `json:"controller_id"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}

	if err := h.registrationSvc.UpdateHealth(r.Context(), tenantID, controllerID); err != nil {
		respondRegistrationError(w, "health_update_failed", err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{
		"acknowledged": true,
	})
}

func (h *InternalHandlers) readAndVerifySignedRequest(w http.ResponseWriter, r *http.Request) (body []byte, tenantID, controllerID uuid.UUID, ok bool) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "failed to read request body")
		return nil, uuid.Nil, uuid.Nil, false
	}

	var req struct {
		TenantID     string `json:"tenant_id"`
		ControllerID string `json:"controller_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return nil, uuid.Nil, uuid.Nil, false
	}

	tenantID, err = uuid.Parse(req.TenantID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid tenant_id")
		return nil, uuid.Nil, uuid.Nil, false
	}
	controllerID, err = uuid.Parse(req.ControllerID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "bad_request", "invalid controller_id")
		return nil, uuid.Nil, uuid.Nil, false
	}

	sig := r.Header.Get(controllerSignatureHeader)
	if sig == "" {
		respondError(w, http.StatusUnauthorized, "unauthenticated", "missing "+controllerSignatureHeader+" header")
		return nil, uuid.Nil, uuid.Nil, false
	}

	if err := h.registrationSvc.VerifyControllerSignature(r.Context(), tenantID, controllerID, body, sig); err != nil {
		var regErr *domain.RegistrationError
		if errors.As(err, &regErr) {
			respondError(w, http.StatusUnauthorized, "unauthenticated", "signature verification failed")
			return nil, uuid.Nil, uuid.Nil, false
		}
		slog.Error("controller signature verification failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return nil, uuid.Nil, uuid.Nil, false
	}

	return body, tenantID, controllerID, true
}

func respondRegistrationError(w http.ResponseWriter, code string, err error) {
	var regErr *domain.RegistrationError
	if errors.As(err, &regErr) {
		respondError(w, http.StatusBadRequest, code, regErr.Error())
		return
	}
	slog.Error("registration request failed", "code", code, "error", err)
	respondError(w, http.StatusInternalServerError, code, "an internal error occurred")
}
