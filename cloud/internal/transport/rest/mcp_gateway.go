/*
Package rest — mcp_gateway.go implements the MCP Identity Gateway HTTP server.

MCPGatewayServer handles MCP protocol traffic (JSON-RPC over HTTP) from AI agents.
It validates Bearer tokens (left-side auth), parses MCP methods, routes to downstream
MCP servers with per-user credentials (right-side auth), and aggregates results.
Served at https://<tenant>-mcpgateway.<domain>/mcp in production.

resolveScope fails closed: a request arriving on any hostname other than the
tenant's Gateway host is rejected with 404. This matters because all routes share
one mux, so POST /mcp is technically reachable on every hostname this process
serves — including the MCP Auth Server host, whose BaseURL would otherwise be
handed to left-side auth as the expected token audience.

HandleProtectedResourceMetadata serves RFC 9728 Protected Resource Metadata.
MCP clients discover which Authorization Server protects this Gateway by either
following the resource_metadata URL in the 401 WWW-Authenticate header, or by
probing /.well-known/oauth-protected-resource on the server's origin.
*/
package rest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
	"github.com/google/uuid"
)

type GatewayAuthenticator interface {
	Authenticate(token string, expectedAudience string) (*gw.AuthResult, error)
}

type GatewayMethodHandler interface {
	HandleMethod(ctx context.Context, rCtx *gw.RequestContext, method string, params json.RawMessage) (any, error)
}

type ConsentCallbackHandler interface {
	HandleCallback(ctx context.Context, code string, state string, scope gw.TenantScope) error
}

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      any           `json:"id"`
	Result  any           `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrNotGatewayHost means the tenant resolved but the request arrived on a
// different service's hostname.
var ErrNotGatewayHost = errors.New("request hostname is not this tenant's MCP Identity Gateway host")

type MCPGatewayServer struct {
	auth           GatewayAuthenticator
	methodHandler  GatewayMethodHandler
	consentHandler ConsentCallbackHandler
	resolver       *TenantHostResolver
	events         *events.EventBus
	logger         *slog.Logger
}

func NewMCPGatewayServer(
	auth GatewayAuthenticator,
	handler GatewayMethodHandler,
	consentHandler ConsentCallbackHandler,
	logger *slog.Logger,
	eventBus *events.EventBus,
	resolver *TenantHostResolver,
) *MCPGatewayServer {
	return &MCPGatewayServer{
		auth:           auth,
		methodHandler:  handler,
		consentHandler: consentHandler,
		resolver:       resolver,
		events:         eventBus,
		logger:         logger,
	}
}

func (s *MCPGatewayServer) resolveScope(r *http.Request) (gw.TenantScope, error) {
	resolved, err := s.resolver.Resolve(r.Context(), r)
	if err != nil {
		return gw.TenantScope{}, err
	}
	if resolved.Service != platform.ServiceGateway {
		return gw.TenantScope{}, ErrNotGatewayHost
	}

	scope := gw.TenantScope{TenantID: resolved.TenantID(), TenantSubdomain: resolved.Subdomain, GatewayURL: resolved.BaseURL}
	if !scope.Valid() {
		return gw.TenantScope{}, ErrTenantHostUnresolved
	}
	return scope, nil
}

func (s *MCPGatewayServer) respondUnresolvedTenant(w http.ResponseWriter, r *http.Request, err error) {
	if s.logger != nil {
		s.logger.Warn("MCP gateway request could not be attributed to a tenant",
			"host", PublicHost(r), "path", r.URL.Path, "error", err)
	}
	writeGatewayError(w, http.StatusNotFound, "this hostname does not serve a Dipole MCP Identity Gateway")
}

func (s *MCPGatewayServer) RegisterGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /mcp", s.HandleMCP)
	mux.HandleFunc("DELETE /mcp", s.HandleSessionDelete)
	mux.HandleFunc("GET /mcp/consent/callback", s.HandleConsentCallback)
	mux.HandleFunc("GET /.well-known/oauth-protected-resource", s.HandleProtectedResourceMetadata)
}

// HandleMCP processes all MCP protocol traffic.
func (s *MCPGatewayServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	scope, err := s.resolveScope(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	token := extractBearerToken(r)
	if token == "" {
		resourceMetadataURL := scope.GatewayURL + "/.well-known/oauth-protected-resource"
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadataURL+`"`)
		s.emitTrace(scope.TenantID, clientIP, r.Header.Get("Origin"), scope.GatewayURL, "Error", "Missing token", "", "")
		writeGatewayError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
		return
	}

	authResult, err := s.auth.Authenticate(token, scope.GatewayURL)
	if err != nil {
		resourceMetadataURL := scope.GatewayURL + "/.well-known/oauth-protected-resource"
		w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadataURL+`", error="invalid_token"`)
		s.emitTrace(scope.TenantID, clientIP, r.Header.Get("Origin"), scope.GatewayURL, "Error", "Invalid token", "", "")
		s.emitLeftSideAuth(r.Context(), scope.TenantID, clientIP, r.Host, "", "", false, err.Error())
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

	s.emitTrace(scope.TenantID, clientIP, r.Header.Get("Origin"), scope.GatewayURL, "Info", "Authenticated", authResult.ClientWorkloadName, authResult.UserID)
	s.emitLeftSideAuth(r.Context(), scope.TenantID, clientIP, r.Host, authResult.UserID, authResult.ClientWorkloadName, true, "")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeGatewayError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var rpcReq JSONRPCRequest
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		writeJSONRPCError(w, nil, -32700, "Parse error")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		writeJSONRPCError(w, rpcReq.ID, -32600, "Invalid Request: jsonrpc must be 2.0")
		return
	}

	rCtx := &gw.RequestContext{
		UserID: authResult.UserID,
		Claims: authResult.Claims,
		Scope:  scope,
	}

	result, err := s.methodHandler.HandleMethod(r.Context(), rCtx, rpcReq.Method, rpcReq.Params)
	if err != nil {
		writeJSONRPCError(w, rpcReq.ID, -32603, err.Error())
		return
	}

	writeJSONRPCResponse(w, rpcReq.ID, result)
}

// HandleSessionDelete terminates an MCP session.
func (s *MCPGatewayServer) HandleSessionDelete(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func writeGatewayError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSONRPCError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &JSONRPCError{Code: code, Message: message},
	})
}

func writeJSONRPCResponse(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *MCPGatewayServer) HandleConsentCallback(w http.ResponseWriter, r *http.Request) {
	scope, err := s.resolveScope(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "missing code or state parameter", http.StatusBadRequest)
		return
	}

	if s.consentHandler == nil {
		http.Error(w, "consent handling not configured", http.StatusInternalServerError)
		return
	}

	if err := s.consentHandler.HandleCallback(r.Context(), code, state, scope); err != nil {
		s.logger.Error("consent callback failed", "error", err)
		http.Error(w, "authorization failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(consentSuccessHTML))
}

const consentSuccessHTML = `<!DOCTYPE html>
<html><head><title>Authorization Successful</title></head>
<body><h1>Authorization successful</h1><p>You can close this window.</p></body>
</html>`

func (s *MCPGatewayServer) HandleProtectedResourceMetadata(w http.ResponseWriter, r *http.Request) {
	scope, err := s.resolveScope(r)
	if err != nil {
		s.respondUnresolvedTenant(w, r, err)
		return
	}

	authServerURL := s.resolver.BaseURLFor(scope.Subdomain(), platform.ServiceMCPAuth)
	if authServerURL == "" {
		writeGatewayError(w, http.StatusInternalServerError, "unable to determine authorization server for this tenant")
		return
	}

	metadata := map[string]interface{}{
		"resource":                 scope.GatewayURL + "/mcp",
		"authorization_servers":    []string{authServerURL},
		"bearer_methods_supported": []string{"header"},
	}
	writeJSON(w, http.StatusOK, metadata)
}

func (s *MCPGatewayServer) emitLeftSideAuth(ctx context.Context, tenantID uuid.UUID, clientIP string, host string, userID string, clientWorkloadName string, allowed bool, denyReason string) {
	if s.events == nil {
		return
	}

	severity := "Info"
	if !allowed {
		severity = "Error"
	}

	clientWL := "Unknown"
	if clientWorkloadName != "" {
		clientWL = clientWorkloadName
	}

	trustProvider := ""
	trustAttested := false
	if userID != "" {
		trustProvider = "User Identity (SSO)"
		trustAttested = allowed
	}

	s.events.EmitSecurity(ctx, events.SecurityEvent{
		TenantID:       tenantID,
		SourceIP:       clientIP,
		Source:         "mcp_gateway",
		EventType:      "access.authorization",
		Severity:       severity,
		Allowed:        allowed,
		DenyReason:     denyReason,
		UserID:         userID,
		ClientWorkload: clientWL,
		ServerWorkload: "MCP Identity Gateway",
		ServerHost:     host,
		ServerPort:     443,
		TrustProvider:  trustProvider,
		TrustAttested:  trustAttested,
	})
}

func (s *MCPGatewayServer) emitTrace(tenantID uuid.UUID, clientIP string, redirectURI string, resourceServer string, severity string, policyDetail string, clientWorkloadName string, userID string) {
	if s.events == nil {
		return
	}

	cwName := clientWorkloadName
	cwStatus := "Identified"
	if cwName == "" {
		cwName = userID
	}
	if cwName == "" {
		cwName = "Unknown"
		cwStatus = "Unknown"
	}

	policyStatus := "Active"
	if severity == "Error" {
		policyStatus = "Unknown"
	}

	s.events.EmitTrace(events.TraceEntry{
		TenantID:       tenantID,
		ClientIP:       clientIP,
		Severity:       severity,
		RedirectURI:    redirectURI,
		ResourceServer: resourceServer,
		AccessPolicy:   events.TracePolicyStatus{Name: policyDetail, Status: policyStatus},
		ClientWorkload: events.TraceWorkloadStatus{Name: cwName, Status: cwStatus},
		ServerWorkload: events.TraceWorkloadStatus{Name: "Dipole MCP Gateway", Status: "Identified"},
	})
}
