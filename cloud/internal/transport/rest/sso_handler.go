/*
sso_handler.go serves SSO authentication endpoints:
  - GET  /api/v1/auth/sso/providers        — public: list available IdPs for login page
  - GET  /api/v1/auth/sso/authorize        — public: redirect to IdP (returns redirect URL)
  - POST /api/v1/auth/sso/callback         — public: exchange code, issue JWT
*/
package rest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type SSOHandler struct {
	idpRepo  repository.IdentityProviderRepository
	ssoSvc   *service.AdminSSOService
	issuer   *identity.IdentityIssuer
	tenantID uuid.UUID
}

func NewSSOHandler(
	idpRepo repository.IdentityProviderRepository,
	ssoSvc *service.AdminSSOService,
	issuer *identity.IdentityIssuer,
	tenantID uuid.UUID,
) *SSOHandler {
	return &SSOHandler{
		idpRepo:  idpRepo,
		ssoSvc:   ssoSvc,
		issuer:   issuer,
		tenantID: tenantID,
	}
}

func (h *SSOHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/sso/providers", h.ListProviders)
	mux.HandleFunc("GET /api/v1/auth/sso/authorize", h.Authorize)
	mux.HandleFunc("POST /api/v1/auth/sso/callback", h.Callback)
}

func (h *SSOHandler) ListProviders(w http.ResponseWriter, r *http.Request) {
	idps, _, err := h.idpRepo.List(r.Context(), h.tenantID, domain.ListOptions{Page: 1, PerPage: 50})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list providers"})
		return
	}

	type providerResponse struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	providers := make([]providerResponse, 0)
	for _, idp := range idps {
		if !idp.IsActive {
			continue
		}
		providers = append(providers, providerResponse{
			ID:   idp.ID.String(),
			Name: idp.Name,
			Type: idp.Type,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"providers": providers})
}

func (h *SSOHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	providerID := r.URL.Query().Get("provider_id")
	if providerID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "provider_id is required"})
		return
	}

	id, err := uuid.Parse(providerID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid provider_id"})
		return
	}

	idp, err := h.idpRepo.GetByID(r.Context(), h.tenantID, id)
	if err != nil || idp == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "identity provider not found"})
		return
	}

	if !idp.IsActive {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "identity provider is inactive"})
		return
	}

	idpConfig := &service.IdentityProviderConfig{
		ID:       idp.ID,
		TenantID: h.tenantID,
		Name:     idp.Name,
		Type:     idp.Type,
		Config:   idp.Config,
	}

	state := uuid.New().String()

	callbackURL := r.URL.Query().Get("callback_url")
	if callbackURL == "" {
		scheme := "https"
		if r.Header.Get("X-Forwarded-Proto") == "http" || r.TLS == nil {
			scheme = "http"
		}
		callbackURL = scheme + "://" + r.Host + "/auth/sso/callback"
	}

	if idp.Type == "oidc_1.0" {
		redirectURL, err := h.ssoSvc.BuildOIDCLoginURL(r.Context(), idpConfig, callbackURL, state)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build SSO URL: " + err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"redirectUrl": redirectURL,
			"state":       state,
		})
		return
	}

	if idp.Type == "saml_2.0" {
		var samlCfg struct {
			SSOURL string `json:"sso_url"`
		}
		if err := json.Unmarshal(idp.Config, &samlCfg); err != nil || samlCfg.SSOURL == "" {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "SAML SSO URL not configured"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"redirectUrl": samlCfg.SSOURL + "?RelayState=" + state,
			"state":       state,
		})
		return
	}

	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported provider type"})
}

func (h *SSOHandler) Callback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProviderID  string `json:"providerId"`
		Code        string `json:"code"`
		CallbackURL string `json:"callbackUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.ProviderID == "" || req.Code == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "providerId and code are required"})
		return
	}

	providerUUID, err := uuid.Parse(req.ProviderID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid providerId"})
		return
	}

	idp, err := h.idpRepo.GetByID(r.Context(), h.tenantID, providerUUID)
	if err != nil || idp == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "identity provider not found"})
		return
	}

	idpConfig := &service.IdentityProviderConfig{
		ID:              idp.ID,
		TenantID:        h.tenantID,
		Name:            idp.Name,
		Type:            idp.Type,
		Config:          idp.Config,
		AutoCreateUsers: true,
	}

	callbackURL := req.CallbackURL
	if callbackURL == "" {
		scheme := "https"
		if r.Header.Get("X-Forwarded-Proto") == "http" || r.TLS == nil {
			scheme = "http"
		}
		callbackURL = scheme + "://" + r.Host + "/auth/sso/callback"
	}

	result, err := h.ssoSvc.HandleOIDCCallback(r.Context(), idpConfig, req.Code, callbackURL)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		Subject:   result.Email,
		Audience:  "dipole-admin",
		Lifetime:  24 * time.Hour,
		Algorithm: "ES256",
		CustomClaims: map[string]interface{}{
			"tenant_id": h.tenantID.String(),
			"user_id":   result.UserID.String(),
			"email":     result.Email,
			"name":      result.Name,
		},
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to issue session token"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"token":     token,
		"expiresAt": time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339),
		"user": map[string]string{
			"id":    result.UserID.String(),
			"email": result.Email,
			"name":  result.Name,
		},
	})
}
