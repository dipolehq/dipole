/*
Package rest — mcp_gateway.go implements the MCP Identity Gateway HTTP server.

MCPGatewayServer handles MCP protocol traffic (JSON-RPC over HTTP) from AI agents.
It validates Bearer tokens (left-side auth), parses MCP methods, routes to downstream
MCP servers with per-user credentials (right-side auth), and aggregates results.
Served at https://<tenant>.mcpgateway.<region>.dipole.io/mcp in production.
*/
package rest

import (
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
	HandleMethod(ctx *gw.RequestContext, method string, params json.RawMessage) (any, error)
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
	auth          GatewayAuthenticator
	methodHandler GatewayMethodHandler
	logger        *slog.Logger
}

func NewMCPGatewayServer(auth GatewayAuthenticator, handler GatewayMethodHandler, logger *slog.Logger) *MCPGatewayServer {
	return &MCPGatewayServer{
		auth:          auth,
		methodHandler: handler,
		logger:        logger,
	}
}

func (s *MCPGatewayServer) RegisterGatewayRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /mcp", s.HandleMCP)
	mux.HandleFunc("DELETE /mcp", s.HandleSessionDelete)
}

// HandleMCP processes all MCP protocol traffic. Validates the Bearer token,
// parses the JSON-RPC request, and routes to the appropriate method handler.
func (s *MCPGatewayServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		writeGatewayError(w, http.StatusUnauthorized, "missing or invalid Authorization header")
		return
	}

	authResult, err := s.auth.Authenticate(token)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		writeGatewayError(w, http.StatusUnauthorized, err.Error())
		return
	}

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

	ctx := &gw.RequestContext{
		UserID: authResult.UserID,
		Claims: authResult.Claims,
	}

	result, err := s.methodHandler.HandleMethod(ctx, rpcReq.Method, rpcReq.Params)
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
