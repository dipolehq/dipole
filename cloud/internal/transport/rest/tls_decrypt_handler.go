/*
tls_decrypt_handler.go serves the TLS Decrypt configuration and Root CA
download endpoints.

GET  /api/v1/tls-decrypt         — returns TLS Decrypt config (enabled, leaf lifetime)
PUT  /api/v1/tls-decrypt         — update leaf certificate lifetime
GET  /api/v1/tls-decrypt/root-ca — download the tenant's Root CA public certificate (PEM)
*/
package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type tlsDecryptService interface {
	GetTenantCACert(ctx context.Context, tenantID uuid.UUID) (string, error)
	IsTlsDecryptEnabled(ctx context.Context, tenantID uuid.UUID) bool
	GetLeafLifetime(ctx context.Context, tenantID uuid.UUID) (int, error)
	SetLeafLifetime(ctx context.Context, tenantID uuid.UUID, lifetimeSeconds int) error
}

type TlsDecryptHandler struct {
	svc tlsDecryptService
}

func NewTlsDecryptHandler(svc tlsDecryptService) *TlsDecryptHandler {
	return &TlsDecryptHandler{svc: svc}
}

func (h *TlsDecryptHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/tls-decrypt", h.GetConfig)
	mux.HandleFunc("PUT /api/v1/tls-decrypt", h.UpdateConfig)
	mux.HandleFunc("GET /api/v1/tls-decrypt/root-ca", h.DownloadRootCA)
}

func (h *TlsDecryptHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	enabled := h.svc.IsTlsDecryptEnabled(r.Context(), tenantID)

	lifetime, err := h.svc.GetLeafLifetime(r.Context(), tenantID)
	if err != nil {
		slog.Error("get leaf lifetime failed", "error", err)
		lifetime = 86400 // default 1 day
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"enabled":             enabled,
		"leafLifetimeSeconds": lifetime,
	})
}

func (h *TlsDecryptHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	var req struct {
		LeafLifetimeSeconds int `json:"leafLifetimeSeconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondBadRequest(w, "invalid request body")
		return
	}

	_, err := h.svc.GetTenantCACert(r.Context(), tenantID)
	if err != nil {
		slog.Error("ensure tenant CA failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to initialize TLS Decrypt")
		return
	}

	if err := h.svc.SetLeafLifetime(r.Context(), tenantID, req.LeafLifetimeSeconds); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"enabled":             true,
		"leafLifetimeSeconds": req.LeafLifetimeSeconds,
	})
}

func (h *TlsDecryptHandler) DownloadRootCA(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if tenantID == uuid.Nil {
		respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	certPEM, err := h.svc.GetTenantCACert(r.Context(), tenantID)
	if err != nil {
		slog.Error("get tenant CA cert failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "failed to retrieve Root CA")
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Header().Set("Content-Disposition", "attachment; filename=\"dipole-tenant-root-ca.pem\"")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(certPEM))
}
