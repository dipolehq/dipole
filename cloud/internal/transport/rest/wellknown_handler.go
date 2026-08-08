/*
Package rest — wellknown_handler.go serves the /.well-known/* discovery
endpoints for every tenant.

These endpoints are shared between TWO different services on TWO different
hostnames, which is why they live here rather than inside either one:

	https://<tenant>-mcp.<domain>/.well-known/oauth-authorization-server
	https://<tenant>-mcp.<domain>/.well-known/openid-configuration
	    → OAuth 2.1 Authorization Server metadata (RFC 8414), issuer = the -mcp host.
	      MCP clients (Claude Desktop) fetch this to discover our OAuth endpoints.

	https://<tenant>-id.<domain>/.well-known/openid-configuration
	    → OIDC discovery for the token issuer, issuer = the -id host.
	      AWS STS / Azure Entra / GCP WIF fetch this to verify tokens we sign.

Serving the wrong document on either host breaks the corresponding integration:
OIDC requires the `issuer` in the document to equal the URL it was fetched
from, and all three cloud federation providers enforce that. Previously one
handler answered both hostnames with the -mcp issuer, so cloud federation
could never verify a token.

The `.mcp.` vs `.id.` split mirrors Aembit's documented model, where the
authorization server and the token issuer are deliberately different hosts.
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

// HandleOAuthMetadata serves OAuth 2.1 Authorization Server metadata. Only
// meaningful on the MCP Authorization Server host.
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

// HandleOpenIDConfiguration serves whichever discovery document matches the
// host it was called on. Per the MCP specification the authorization server
// returns identical content for oauth-authorization-server and
// openid-configuration, so the -mcp host answers with AS metadata.
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

// HandleJWKS returns the public signing keys. Deliberately host-agnostic: the
// signing key is shared platform-wide (per-tenant scoping comes from the `iss`
// claim, not from separate keys), and this is public key material, so there is
// nothing to gate. Serving it on any host also means a relying party can never
// fail to fetch it.
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

// OAuthAuthorizationServerMetadata builds the RFC 8414 metadata document for an
// MCP Authorization Server reachable at serverURL.
//
// Only capabilities we actually implement are advertised — advertising anything
// else causes clients to attempt flows we then reject.
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
