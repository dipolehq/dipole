/*
Routes:

	GET /.well-known/oauth-authorization-server
	GET /.well-known/openid-configuration
	GET /.well-known/openid-configuration/jwks
*/
package rest

import (
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type WellKnownHandler struct {
	issuer   *identity.IdentityIssuer
	resolver *TenantHostResolver
	logger   *slog.Logger
}

func NewWellKnownHandler(issuer *identity.IdentityIssuer, resolver *TenantHostResolver, logger *slog.Logger) *WellKnownHandler {
	return &WellKnownHandler{issuer: issuer, resolver: resolver, logger: logger}
}

func (h *WellKnownHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.HandleOAuthMetadata)
	mux.HandleFunc("GET /.well-known/openid-configuration", h.HandleOpenIDConfiguration)
	mux.HandleFunc("GET /.well-known/openid-configuration/jwks", h.HandleJWKS)
}

func (h *WellKnownHandler) HandleOAuthMetadata(w http.ResponseWriter, r *http.Request) {
	resolved, err := h.resolver.Resolve(r.Context(), r)
	if err != nil {
		h.respondUnresolved(w, r, err)
		return
	}
	if resolved.Service != platform.ServiceMCPAuth {
		writeJSONError(w, http.StatusNotFound, "not_found",
			"OAuth authorization server metadata is served on the MCP authorization server host")
		return
	}

	writeJSON(w, http.StatusOK, OAuthAuthorizationServerMetadata(resolved.BaseURL))
}

func (h *WellKnownHandler) HandleOpenIDConfiguration(w http.ResponseWriter, r *http.Request) {
	resolved, err := h.resolver.Resolve(r.Context(), r)
	if err != nil {
		h.respondUnresolved(w, r, err)
		return
	}

	switch resolved.Service {
	case platform.ServiceIdentity:
		writeJSON(w, http.StatusOK, h.issuer.DiscoveryDocument(resolved.BaseURL))
	case platform.ServiceMCPAuth:
		writeJSON(w, http.StatusOK, OAuthAuthorizationServerMetadata(resolved.BaseURL))
	default:
		writeJSONError(w, http.StatusNotFound, "not_found", "no discovery document is served on this host")
	}
}

func (h *WellKnownHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.issuer.JWKS())
}

func (h *WellKnownHandler) respondUnresolved(w http.ResponseWriter, r *http.Request, err error) {
	if h.logger != nil {
		h.logger.Warn("discovery request could not be attributed to a tenant",
			"host", PublicHost(r), "path", r.URL.Path, "error", err)
	}
	writeJSONError(w, http.StatusNotFound, "not_found", "unknown tenant for this hostname")
}

func OAuthAuthorizationServerMetadata(serverURL string) map[string]any {
	return map[string]any{
		"issuer":                                serverURL,
		"authorization_endpoint":                serverURL + "/connect/authorize",
		"token_endpoint":                        serverURL + "/connect/token",
		"jwks_uri":                              serverURL + "/.well-known/openid-configuration/jwks",
		"registration_endpoint":                 serverURL + "/register",
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"response_types_supported":              []string{"code"},
		"response_modes_supported":              []string{"query"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"none"},
	}
}
