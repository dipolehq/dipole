/*
method_handler.go implements MCP method routing for the Identity Gateway.

MethodHandler dispatches MCP JSON-RPC methods (tools/list, tools/call,
resources/list, resources/read, ping) to the appropriate handler. It aggregates
tools from multiple MCP servers with name prefixes, routes tool calls to the
correct server, and handles resource discovery without prefixes.
*/
package mcp_gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/events"
	"github.com/google/uuid"
)

const toolPrefixSeparator = "__"

var ErrMethodNotFound = errors.New("method not found")

type AuthEventEmitterInterface interface {
	Emit(ctx context.Context, eval *events.EvaluationContext) error
}

func sanitizeToolName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	var b strings.Builder
	prevUnderscore := false
	for _, c := range name {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			b.WriteRune(c)
			prevUnderscore = false
		} else if c == '_' {
			if !prevUnderscore {
				b.WriteRune(c)
			}
			prevUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

func toolPrefix(server ServerInfo) string {
	short := strings.ReplaceAll(server.ID.String(), "-", "")[:8]
	return sanitizeToolName(server.Name) + "_" + short
}

type MCPTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
	Annotations map[string]any `json:"annotations,omitempty"`
}

type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

type MCPServerClient interface {
	ListTools(ctx context.Context, token string) ([]MCPTool, error)
	CallTool(ctx context.Context, token string, toolName string, args json.RawMessage) (any, error)
	ListResources(ctx context.Context, token string) ([]MCPResource, error)
	ReadResource(ctx context.Context, token string, uri string) (*MCPResourceContent, error)
}

type RequestContext struct {
	UserID string
	Claims map[string]any
	Scope  TenantScope
}

type MethodHandler struct {
	rightAuth   *RightSideAuth
	clients     MCPClientFactory
	emitter     *GatewayEventEmitter
	authEmitter AuthEventEmitterInterface
	logger      *slog.Logger
}

func NewMethodHandler(rightAuth *RightSideAuth, clients MCPClientFactory, emitter *GatewayEventEmitter, authEmitter AuthEventEmitterInterface, logger *slog.Logger) *MethodHandler {
	return &MethodHandler{
		rightAuth:   rightAuth,
		clients:     clients,
		emitter:     emitter,
		authEmitter: authEmitter,
		logger:      logger,
	}
}

func (h *MethodHandler) HandleMethod(ctx context.Context, rCtx *RequestContext, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return h.handleInitialize()
	case "ping":
		return h.handlePing()
	case "tools/list":
		return h.handleToolsList(ctx, rCtx)
	case "tools/call":
		return h.handleToolsCall(ctx, rCtx, params)
	case "resources/list":
		return h.handleResourcesList(ctx, rCtx)
	case "resources/read":
		return h.handleResourcesRead(ctx, rCtx, params)
	default:
		return nil, ErrMethodNotFound
	}
}

func (h *MethodHandler) handlePing() (any, error) {
	return map[string]any{}, nil
}

func (h *MethodHandler) handleInitialize() (any, error) {
	return map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]any{
			"name":    "Dipole MCP Identity Gateway",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"tools":     map[string]any{},
			"resources": map[string]any{},
		},
	}, nil
}

func (h *MethodHandler) handleToolsList(ctx context.Context, rCtx *RequestContext) (any, error) {
	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, rCtx.Scope)
	if err != nil {
		return nil, err
	}

	allTools := make([]MCPTool, 0)
	for _, cred := range creds {
		client := h.clients.ClientFor(cred.ServerWorkload)

		tools, err := client.ListTools(ctx, cred.AccessToken)
		if err != nil {
			continue
		}

		for _, tool := range tools {
			tool.Name = toolPrefix(cred.ServerWorkload) + toolPrefixSeparator + tool.Name
			allTools = append(allTools, tool)
		}
	}

	return map[string]any{"tools": allTools}, nil
}

func (h *MethodHandler) handleToolsCall(ctx context.Context, rCtx *RequestContext, params json.RawMessage) (any, error) {
	var req struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid tools/call params")
	}

	serverName, toolName, err := parsePrefixedToolName(req.Name)
	if err != nil {
		return nil, err
	}

	cred, err := h.rightAuth.GetCredentialByServerName(ctx, rCtx.UserID, rCtx.Scope, serverName)
	if err != nil {
		h.emitRightSideAuthEvent(ctx, rCtx, serverName, toolName, false, err.Error())
		return nil, err
	}

	client := h.clients.ClientFor(cred.ServerWorkload)

	result, err := client.CallTool(ctx, cred.AccessToken, toolName, req.Arguments)
	if err != nil {
		if h.emitter != nil {
			h.emitter.EmitResponse(rCtx.Scope.TenantID, rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID, 500)
		}
		return nil, err
	}

	if h.emitter != nil {
		h.emitter.EmitRequest(rCtx.Scope.TenantID, rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID)
		h.emitter.EmitResponse(rCtx.Scope.TenantID, rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID, 200)
	}

	h.emitRightSideAuthEvent(ctx, rCtx, cred.ServerWorkload.Name, toolName, true, "")

	return result, nil
}

func (h *MethodHandler) handleResourcesList(ctx context.Context, rCtx *RequestContext) (any, error) {
	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, rCtx.Scope)
	if err != nil {
		return nil, err
	}

	allResources := make([]MCPResource, 0)
	for _, cred := range creds {
		client := h.clients.ClientFor(cred.ServerWorkload)

		resources, err := client.ListResources(ctx, cred.AccessToken)
		if err != nil {
			continue
		}

		allResources = append(allResources, resources...)
	}

	return map[string]any{"resources": allResources}, nil
}

func (h *MethodHandler) handleResourcesRead(ctx context.Context, rCtx *RequestContext, params json.RawMessage) (any, error) {
	var req struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid resources/read params")
	}

	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, rCtx.Scope)
	if err != nil {
		return nil, err
	}

	for _, cred := range creds {
		client := h.clients.ClientFor(cred.ServerWorkload)

		content, err := client.ReadResource(ctx, cred.AccessToken, req.URI)
		if err == nil && content != nil {
			if h.emitter != nil {
				h.emitter.EmitRequest(rCtx.Scope.TenantID, rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID)
				h.emitter.EmitResponse(rCtx.Scope.TenantID, rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID, 200)
			}
			return content, nil
		}
	}

	return nil, fmt.Errorf("resource not found: %s", req.URI)
}

func parsePrefixedToolName(prefixed string) (serverName, toolName string, err error) {
	idx := strings.Index(prefixed, toolPrefixSeparator)
	if idx < 0 {
		return "", "", fmt.Errorf("tool name missing server prefix: %s", prefixed)
	}
	serverName = prefixed[:idx]
	toolName = prefixed[idx+len(toolPrefixSeparator):]
	if serverName == "" || toolName == "" {
		return "", "", fmt.Errorf("invalid prefixed tool name: %s", prefixed)
	}
	return serverName, toolName, nil
}

func (h *MethodHandler) emitRightSideAuthEvent(ctx context.Context, rCtx *RequestContext, serverName string, toolName string, allowed bool, denyReason string) {
	if h.authEmitter == nil {
		return
	}

	result := &domain.EvaluationResult{Allowed: allowed, DenyReason: denyReason}

	// The "client" in right-side auth is the Gateway itself acting on behalf of the user.
	clientWL := &domain.ClientWorkload{Name: "MCP Identity Gateway"}
	serverWL := &domain.ServerWorkload{Name: serverName}

	evalCtx := &events.EvaluationContext{
		TenantID:       rCtx.Scope.TenantID,
		SourceIP:       rCtx.UserID,
		TargetHost:     serverName,
		TargetPort:     443,
		Result:         result,
		ClientWorkload: clientWL,
		ServerWorkload: serverWL,
		Source:         "mcp_gateway",
	}

	// Include user + tool info in the payload via a synthetic policy
	if rCtx.UserID != "" || toolName != "" {
		policyName := "Gateway → " + serverName
		evalCtx.Policy = &domain.AccessPolicy{
			ID:   uuid.New(),
			Name: policyName,
		}
	}

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := h.authEmitter.Emit(asyncCtx, evalCtx); err != nil {
			if h.logger != nil {
				h.logger.Error("failed to emit MCP access log", "error", err, "server", serverName, "user", rCtx.UserID, "tool", toolName)
			}
		}
	}()
}
