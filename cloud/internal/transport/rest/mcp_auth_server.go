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
	"errors"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	"github.com/dipolehq/dipole/cloud/internal/service/mcp"
	"github.com/google/uuid"
)

type MCPAuthServer struct {
	issuer *identity.IdentityIssuer
	dcr    *mcp.DCRService
	config MCPAuthConfig
	logger *slog.Logger
}

type MCPAuthConfig struct {
	TenantID        uuid.UUID
	TenantSubdomain string
	Region          string
	Domain          string
}

func NewMCPAuthServer(issuer *identity.IdentityIssuer, dcr *mcp.DCRService, cfg MCPAuthConfig, logger *slog.Logger) *MCPAuthServer {
	return &MCPAuthServer{
		issuer: issuer,
		dcr:    dcr,
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
	mux.HandleFunc("POST /register", s.HandleRegister)
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

// HandleRegister implements Dynamic Client Registration (RFC 7591).
// MCP clients POST their metadata to receive a unique client_id.
func (s *MCPAuthServer) HandleRegister(w http.ResponseWriter, r *http.Request) {
	tenantID, err := s.extractTenantID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing or invalid tenant")
		return
	}

	var req mcp.DCRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "malformed JSON body")
		return
	}

	resp, err := s.dcr.Register(r.Context(), tenantID, &req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// extractTenantID resolves the tenant from the request context or header.
// In production this comes from subdomain routing; in dev from X-Tenant-ID header.
func (s *MCPAuthServer) extractTenantID(r *http.Request) (uuid.UUID, error) {
	if h := r.Header.Get("X-Tenant-ID"); h != "" {
		return uuid.Parse(h)
	}
	if s.config.TenantID != uuid.Nil {
		return s.config.TenantID, nil
	}
	return uuid.Nil, errors.New("tenant not resolved")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, errCode, description string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": description,
	})
}
