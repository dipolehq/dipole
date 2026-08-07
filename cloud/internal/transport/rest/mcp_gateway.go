/*
Package rest — mcp_gateway.go implements the MCP Identity Gateway HTTP server.

MCPGatewayServer handles MCP protocol traffic (JSON-RPC over HTTP) from AI agents.
It validates Bearer tokens (left-side auth), parses MCP methods, routes to downstream
MCP servers with per-user credentials (right-side auth), and aggregates results.
Served at https://<tenant>.mcpgateway.dipolehq.com/mcp in production.
*/
package rest

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	gw "github.com/dipolehq/dipole/cloud/internal/service/mcp_gateway"
)

type GatewayAuthenticator interface {
	Authenticate(token string) (*gw.AuthResult, error)
}

type GatewayMethodHandler interface {
	HandleMethod(ctx context.Context, rCtx *gw.RequestContext, method string, params json.RawMessage) (any, error)
}

type ConsentCallbackHandler interface {
	HandleCallback(ctx context.Context, code string, state string) error
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

type MCPGatewayServer struct {
	auth           GatewayAuthenticator
	methodHandler  GatewayMethodHandler
	consentHandler ConsentCallbackHandler
	tracingBus     *gw.TracingBus
	logger         *slog.Logger
}

func NewMCPGatewayServer(auth GatewayAuthenticator, handler GatewayMethodHandler, consentHandler ConsentCallbackHandler, logger *slog.Logger, tracingBus *gw.TracingBus) *MCPGatewayServer {
	return &MCPGatewayServer{
		auth:           auth,
		methodHandler:  handler,
		consentHandler: consentHandler,
		tracingBus:     tracingBus,
		logger:         logger,
	}
}

func (s *MCPGatewayServer) RegisterGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /mcp", s.HandleMCP)
	mux.HandleFunc("DELETE /mcp", s.HandleSessionDelete)
	mux.HandleFunc("GET /mcp/consent/callback", s.HandleConsentCallback)
}

// HandleMCP processes all MCP protocol traffic. Validates the Bearer token,
// parses the JSON-RPC request, and routes to the appropriate method handler.
func (s *MCPGatewayServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		s.emitTracing(r, "", "", "Error", "Missing token")
		writeGatewayError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
		return
	}

	authResult, err := s.auth.Authenticate(token)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		s.emitTracing(r, "", "", "Error", "Invalid token: "+err.Error())
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

	s.emitTracing(r, authResult.UserID, "", "Info", "Authenticated")

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
	}

	result, err := s.methodHandler.HandleMethod(r.Context(), rCtx, rpcReq.Method, rpcReq.Params)
	if err != nil {
		writeJSONRPCError(w, rpcReq.ID, -32603, err.Error())
		return
	}

	writeJSONRPCResponse(w, rpcReq.ID, result)
}

// HandleSessionDelete terminates an MCP session. Responds 204 (idempotent).
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

	if err := s.consentHandler.HandleCallback(r.Context(), code, state); err != nil {
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

func (s *MCPGatewayServer) emitTracing(r *http.Request, userID string, resource string, severity string, detail string) {
	if s.tracingBus == nil {
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	entry := gw.TracingEntry{
		ClientIP:       clientIP,
		Severity:       severity,
		RedirectURI:    r.Header.Get("Origin"),
		ResourceServer: resource,
		AccessPolicy:   gw.TracingPolicy{Name: detail, Status: severity},
		ClientWorkload: gw.TracingWL{Name: userID, Status: "Identified"},
		ServerWorkload: gw.TracingWL{Name: resource, Status: "Unknown"},
	}

	if userID == "" {
		entry.ClientWorkload = gw.TracingWL{Name: "Unknown", Status: "Unknown"}
	}

	s.tracingBus.Emit(entry)
}
