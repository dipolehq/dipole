/*
Package rest — mcp_server.go implements the Dipole MCP Server endpoint.

DipoleMCPServer exposes event logs (audit, authorization, workload) as read-only
MCP tools for AI agents. Authenticated via API Token Bearer. Handles JSON-RPC
methods: initialize, ping, tools/list, tools/call, resources/list, resources/read.
Served at POST /mcp on the .mcp. subdomain alongside the MCP Auth Server.
*/
package rest

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type MCPToolHandler interface {
	ListTools() any
	CallTool(name string, args map[string]any, tenantID uuid.UUID, resourceSetID *uuid.UUID) (any, error)
}

type MCPResourceHandler interface {
	ListResources() any
	ReadResource(uri string) (any, error)
}

type TenantAuthenticator interface {
	ValidateAPIToken(token string) (uuid.UUID, error)
}

type DipoleMCPServer struct {
	tools     MCPToolHandler
	resources MCPResourceHandler
	auth      TenantAuthenticator
	logger    *slog.Logger
}

func NewDipoleMCPServer(tools MCPToolHandler, resources MCPResourceHandler, auth TenantAuthenticator, logger *slog.Logger) *DipoleMCPServer {
	return &DipoleMCPServer{
		tools:     tools,
		resources: resources,
		auth:      auth,
		logger:    logger,
	}
}

func (s *DipoleMCPServer) RegisterMCPServerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /mcp/server", s.HandleMCP)
}

func (s *DipoleMCPServer) HandleMCP(w http.ResponseWriter, r *http.Request) {
	token := extractMCPServerToken(r)
	if token == "" {
		writeMCPServerError(w, nil, -32000, "authentication required: missing Bearer token")
		return
	}

	tenantID, err := s.auth.ValidateAPIToken(token)
	if err != nil {
		writeMCPServerError(w, nil, -32000, "authentication failed: invalid token")
		return
	}

	var resourceSetID *uuid.UUID
	if rsHeader := r.Header.Get("X-Dipole-ResourceSet"); rsHeader != "" {
		if parsed, err := uuid.Parse(rsHeader); err == nil {
			resourceSetID = &parsed
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeMCPServerError(w, nil, -32700, "failed to read request body")
		return
	}

	var rpcReq struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      any             `json:"id"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params,omitempty"`
	}
	if err := json.Unmarshal(body, &rpcReq); err != nil {
		writeMCPServerError(w, nil, -32700, "Parse error")
		return
	}

	if rpcReq.JSONRPC != "2.0" {
		writeMCPServerError(w, rpcReq.ID, -32600, "Invalid Request: jsonrpc must be 2.0")
		return
	}

	var result any
	switch rpcReq.Method {
	case "initialize":
		result = s.handleInitialize()
	case "ping":
		result = map[string]any{}
	case "tools/list":
		result = s.tools.ListTools()
	case "tools/call":
		result, err = s.handleToolsCall(rpcReq.Params, tenantID, resourceSetID)
	case "resources/list":
		result = s.resources.ListResources()
	case "resources/read":
		result, err = s.handleResourcesRead(rpcReq.Params)
	default:
		writeMCPServerError(w, rpcReq.ID, -32601, "Method not found: "+rpcReq.Method)
		return
	}

	if err != nil {
		writeMCPServerError(w, rpcReq.ID, -32603, err.Error())
		return
	}

	writeMCPServerResponse(w, rpcReq.ID, result)
}

func (s *DipoleMCPServer) handleInitialize() any {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]any{
			"name":    "Dipole MCP Server",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
	}
}

func (s *DipoleMCPServer) handleToolsCall(params json.RawMessage, tenantID uuid.UUID, resourceSetID *uuid.UUID) (any, error) {
	var req struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, err
	}
	return s.tools.CallTool(req.Name, req.Arguments, tenantID, resourceSetID)
}

func (s *DipoleMCPServer) handleResourcesRead(params json.RawMessage) (any, error) {
	var req struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, err
	}
	return s.resources.ReadResource(req.URI)
}

func extractMCPServerToken(r *http.Request) string {
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

func writeMCPServerError(w http.ResponseWriter, id any, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	})
}

func writeMCPServerResponse(w http.ResponseWriter, id any, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	})
}

// CompareTokenHash provides timing-safe comparison for API token validation.
func CompareTokenHash(provided, stored string) bool {
	return subtle.ConstantTimeCompare([]byte(provided), []byte(stored)) == 1
}
