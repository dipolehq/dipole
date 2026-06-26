/*
Package rest provides HTTP transport handlers for the Cloud service.

MCPAuthServer implements the OAuth 2.1 Authorization Server endpoints for
MCP client authorization. Served on the <tenant>.mcp.<region>.dipole.io
subdomain. Handles: OAuth metadata discovery, OIDC discovery, JWKS, DCR,
authorization, and token exchange.
*/
package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type MCPAuthServer struct {
	issuer *identity.IdentityIssuer
	config MCPAuthConfig
	logger *slog.Logger
}

type MCPAuthConfig struct {
	TenantSubdomain string
	Region          string
	Domain          string
}

func NewMCPAuthServer(issuer *identity.IdentityIssuer, cfg MCPAuthConfig, logger *slog.Logger) *MCPAuthServer {
	return &MCPAuthServer{
		issuer: issuer,
		config: cfg,
		logger: logger,
	}
}

// serverURL returns the MCP Authorization Server base URL (<tenant>.mcp.<region>.<domain>).
func (s *MCPAuthServer) serverURL() string {
	return "https://" + s.config.TenantSubdomain + ".mcp." + s.config.Region + "." + s.config.Domain
}

func (s *MCPAuthServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", s.HandleMetadata)
	mux.HandleFunc("GET /.well-known/openid-configuration", s.HandleMetadata)
	mux.HandleFunc("GET /.well-known/openid-configuration/jwks", s.HandleJWKS)
}

// HandleMetadata serves OAuth 2.1 metadata and OIDC discovery.
// Both /.well-known/oauth-authorization-server and /.well-known/openid-configuration
// return the same response per the MCP specification.
func (s *MCPAuthServer) HandleMetadata(w http.ResponseWriter, r *http.Request) {
	serverURL := s.serverURL()

	metadata := map[string]any{
		"issuer":                           serverURL,
		"authorization_endpoint":           serverURL + "/connect/authorize",
		"token_endpoint":                   serverURL + "/connect/token",
		"jwks_uri":                         serverURL + "/.well-known/openid-configuration/jwks",
		"registration_endpoint":            serverURL + "/register",
		"scopes_supported":                 []string{"openid", "profile", "email"},
		"response_types_supported":         []string{"code", "token", "id_token", "id_token token", "code id_token", "code token", "code id_token token"},
		"response_modes_supported":         []string{"form_post", "query", "fragment"},
		"grant_types_supported":            []string{"authorization_code"},
		"code_challenge_methods_supported": []string{"plain", "S256"},
	}

	writeJSON(w, http.StatusOK, metadata)
}

// HandleJWKS returns the public signing keys used to verify tokens issued by
// this authorization server. MCP servers fetch these to validate access tokens.
func (s *MCPAuthServer) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	jwks := s.issuer.JWKS()
	writeJSON(w, http.StatusOK, jwks)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
