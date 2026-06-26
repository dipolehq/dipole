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
	"net/url"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	"github.com/dipolehq/dipole/cloud/internal/service/mcp"
	"github.com/google/uuid"
)

type MCPAuthServer struct {
	issuer      *identity.IdentityIssuer
	dcr         *mcp.DCRService
	authorize   *mcp.AuthorizeService
	tokens      *mcp.TokenService
	idpCallback *mcp.IdPCallbackService
	config      MCPAuthConfig
	logger      *slog.Logger
}

type MCPAuthConfig struct {
	TenantID        uuid.UUID
	TenantSubdomain string
	Region          string
	Domain          string
}

func NewMCPAuthServer(issuer *identity.IdentityIssuer, dcr *mcp.DCRService, authz *mcp.AuthorizeService, tokens *mcp.TokenService, idpCallback *mcp.IdPCallbackService, cfg MCPAuthConfig, logger *slog.Logger) *MCPAuthServer {
	return &MCPAuthServer{
		issuer:      issuer,
		dcr:         dcr,
		authorize:   authz,
		tokens:      tokens,
		idpCallback: idpCallback,
		config:      cfg,
		logger:      logger,
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
	mux.HandleFunc("GET /connect/authorize", s.HandleAuthorize)
	mux.HandleFunc("POST /connect/token", s.HandleToken)
	mux.HandleFunc("GET /callback/oidc", s.HandleOIDCCallback)
	mux.HandleFunc("POST /callback/saml", s.HandleSAMLCallback)
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

// HandleAuthorize implements the OAuth 2.1 authorization endpoint with PKCE.
// Validates the request, finds the matching Access Policy, evaluates conditions,
// and redirects with an authorization code.
func (s *MCPAuthServer) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	tenantID, err := s.extractTenantID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing or invalid tenant")
		return
	}

	q := r.URL.Query()
	req := &mcp.AuthorizeRequest{
		ClientID:            q.Get("client_id"),
		RedirectURI:         q.Get("redirect_uri"),
		ResponseType:        q.Get("response_type"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		Scope:               q.Get("scope"),
		State:               q.Get("state"),
		Resource:            q.Get("resource"),
	}

	if req.ResponseType != "code" {
		writeJSONError(w, http.StatusBadRequest, "unsupported_response_type", "response_type must be code")
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	result, err := s.authorize.Authorize(r.Context(), tenantID, req, clientIP)
	if err != nil {
		if errors.Is(err, mcp.ErrClientNotFound) || errors.Is(err, mcp.ErrRedirectURIMismatch) {
			writeJSONError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if req.RedirectURI != "" {
			redirectWithError(w, r, req.RedirectURI, "access_denied", err.Error(), req.State)
			return
		}
		writeJSONError(w, http.StatusBadRequest, "access_denied", err.Error())
		return
	}

	if result.IdPRedirect != "" {
		http.Redirect(w, r, result.IdPRedirect, http.StatusFound)
		return
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// HandleToken implements the OAuth 2.1 token endpoint.
// Supports authorization_code (with PKCE) and refresh_token grant types.
func (s *MCPAuthServer) HandleToken(w http.ResponseWriter, r *http.Request) {
	tenantID, err := s.extractTenantID(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing or invalid tenant")
		return
	}

	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "malformed form body")
		return
	}

	req := &mcp.TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		ClientID:     r.FormValue("client_id"),
		CodeVerifier: r.FormValue("code_verifier"),
		RefreshToken: r.FormValue("refresh_token"),
	}

	resp, err := s.tokens.Exchange(req, tenantID)
	if err != nil {
		errCode := "invalid_grant"
		if err.Error() == "unsupported_grant_type" {
			errCode = "unsupported_grant_type"
		}
		writeJSONError(w, http.StatusBadRequest, errCode, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// HandleOIDCCallback processes the IdP redirect after OIDC user authentication.
// Exchanges the IdP code for an id_token, validates via Trust Provider, and
// issues an MCP authorization code back to the original MCP client.
func (s *MCPAuthServer) HandleOIDCCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing code or state")
		return
	}

	result, err := s.idpCallback.HandleOIDCCallback(r.Context(), code, state)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "access_denied", err.Error())
		return
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
}

// HandleSAMLCallback processes the IdP POST after SAML user authentication.
// Validates the SAML assertion via Trust Provider and issues an MCP authorization code.
func (s *MCPAuthServer) HandleSAMLCallback(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "malformed form body")
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	relayState := r.FormValue("RelayState")

	if samlResponse == "" || relayState == "" {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "missing SAMLResponse or RelayState")
		return
	}

	result, err := s.idpCallback.HandleSAMLCallback(r.Context(), samlResponse, relayState)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "access_denied", err.Error())
		return
	}

	http.Redirect(w, r, result.RedirectURL, http.StatusFound)
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

func redirectWithError(w http.ResponseWriter, r *http.Request, redirectURI, errCode, description, state string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, errCode, description)
		return
	}
	q := u.Query()
	q.Set("error", errCode)
	q.Set("error_description", description)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}
