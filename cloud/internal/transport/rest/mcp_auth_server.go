/*
Routes:

	POST /register
	GET  /connect/authorize
	POST /connect/token
	GET  /callback/oidc
	POST /callback/saml
*/
package rest

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/service/mcp"
)

type MCPAuthServer struct {
	dcr         *mcp.DCRService
	authorize   *mcp.AuthorizeService
	tokens      *mcp.TokenService
	idpCallback *mcp.IdPCallbackService
	resolver    *TenantHostResolver
	logger      *slog.Logger
}

func NewMCPAuthServer(dcr *mcp.DCRService, authz *mcp.AuthorizeService, tokens *mcp.TokenService, idpCallback *mcp.IdPCallbackService, resolver *TenantHostResolver, logger *slog.Logger) *MCPAuthServer {
	return &MCPAuthServer{
		dcr:         dcr,
		authorize:   authz,
		tokens:      tokens,
		idpCallback: idpCallback,
		resolver:    resolver,
		logger:      logger,
	}
}

func (s *MCPAuthServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /register", s.HandleRegister)
	mux.HandleFunc("GET /connect/authorize", s.HandleAuthorize)
	mux.HandleFunc("POST /connect/token", s.HandleToken)

	mux.HandleFunc("GET "+platform.OIDCCallbackPath, s.HandleOIDCCallback)
	mux.HandleFunc("POST "+platform.SAMLCallbackPath, s.HandleSAMLCallback)
}

func (s *MCPAuthServer) resolveMCPHost(r *http.Request) (*ResolvedTenantHost, error) {
	resolved, err := s.resolver.Resolve(r.Context(), r)
	if err != nil {
		return nil, err
	}
	if resolved.Service != platform.ServiceMCPAuth {
		return nil, ErrTenantHostUnresolved
	}
	return resolved, nil
}

func (s *MCPAuthServer) respondUnresolvedTenant(w http.ResponseWriter, r *http.Request, err error) {
	if s.logger != nil {
		s.logger.Warn("MCP auth request could not be attributed to a tenant",
			"host", PublicHost(r), "path", r.URL.Path, "error", err)
	}
	writeJSONError(w, http.StatusBadRequest, "invalid_request",
		"this hostname does not identify a Dipole tenant")
}

func (s *MCPAuthServer) HandleRegister(w http.ResponseWriter, r *http.Request) {
	resolved, err := s.resolveMCPHost(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}
	tenantID := resolved.TenantID()

	var req mcp.DCRRequest
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "failed to read request body")
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		if s.logger != nil {
			s.logger.Warn("DCR request JSON decode failed", "error", err, "body", string(body), "content_type", r.Header.Get("Content-Type"))
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "malformed JSON body")
		return
	}

	resp, err := s.dcr.Register(r.Context(), tenantID, &req)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("DCR registration failed", "error", err, "redirect_uris", req.RedirectURIs, "grant_types", req.GrantTypes)
		}
		writeJSONError(w, http.StatusBadRequest, "invalid_client_metadata", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *MCPAuthServer) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	resolved, err := s.resolveMCPHost(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}
	tenantID := resolved.TenantID()

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

		ServerBaseURL: resolved.BaseURL,
	}

	if req.ResponseType != "code" {
		writeJSONError(w, http.StatusBadRequest, "unsupported_response_type", "response_type must be code")
		return
	}

	clientIP := clientIP(r)

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

func (s *MCPAuthServer) HandleToken(w http.ResponseWriter, r *http.Request) {
	resolved, err := s.resolveMCPHost(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}
	tenantID := resolved.TenantID()

	issuerURL := s.resolver.BaseURLFor(resolved.Subdomain, platform.ServiceIdentity)
	if issuerURL == "" {
		writeJSONError(w, http.StatusInternalServerError, "server_error", "unable to determine token issuer for this tenant")
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

	resp, err := s.tokens.Exchange(r.Context(), req, tenantID, issuerURL)
	if err != nil {
		errCode := "invalid_grant"
		if err.Error() == "unsupported_grant_type" {
			errCode = "unsupported_grant_type"
		}
		writeJSONError(w, http.StatusBadRequest, errCode, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, resp)
}

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
