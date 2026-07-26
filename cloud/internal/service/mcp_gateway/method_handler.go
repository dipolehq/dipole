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
	"strings"

	"github.com/google/uuid"
)

const toolPrefixSeparator = "__"

var ErrMethodNotFound = errors.New("method not found")

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
}

type MethodHandler struct {
	rightAuth *RightSideAuth
	clients   map[string]MCPServerClient
	tenantID  uuid.UUID
	emitter   *GatewayEventEmitter
}

func NewMethodHandler(rightAuth *RightSideAuth, clients map[string]MCPServerClient, tenantID uuid.UUID, emitter *GatewayEventEmitter) *MethodHandler {
	return &MethodHandler{
		rightAuth: rightAuth,
		clients:   clients,
		tenantID:  tenantID,
		emitter:   emitter,
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
	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, h.tenantID)
	if err != nil {
		return nil, err
	}

	var allTools []MCPTool
	for _, cred := range creds {
		client, ok := h.clients[cred.ServerWorkload.Name]
		if !ok {
			continue
		}

		tools, err := client.ListTools(ctx, cred.AccessToken)
		if err != nil {
			continue
		}

		for _, tool := range tools {
			tool.Name = cred.ServerWorkload.Name + toolPrefixSeparator + tool.Name
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

	client, ok := h.clients[serverName]
	if !ok {
		return nil, fmt.Errorf("unknown MCP server: %s", serverName)
	}

	cred, err := h.rightAuth.GetCredentialByServerName(ctx, rCtx.UserID, h.tenantID, serverName)
	if err != nil {
		return nil, err
	}

	result, err := client.CallTool(ctx, cred.AccessToken, toolName, req.Arguments)
	if err != nil {
		if h.emitter != nil {
			h.emitter.EmitResponse(rCtx.UserID, "MCP Client", serverName, cred.ServerWorkload.ID, 500)
		}
		return nil, err
	}

	if h.emitter != nil {
		h.emitter.EmitRequest(rCtx.UserID, "MCP Client", serverName, cred.ServerWorkload.ID)
		h.emitter.EmitResponse(rCtx.UserID, "MCP Client", serverName, cred.ServerWorkload.ID, 200)
	}

	return result, nil
}

func (h *MethodHandler) handleResourcesList(ctx context.Context, rCtx *RequestContext) (any, error) {
	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, h.tenantID)
	if err != nil {
		return nil, err
	}

	var allResources []MCPResource
	for _, cred := range creds {
		client, ok := h.clients[cred.ServerWorkload.Name]
		if !ok {
			continue
		}

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

	creds, err := h.rightAuth.GetAllServerCredentials(ctx, rCtx.UserID, h.tenantID)
	if err != nil {
		return nil, err
	}

	for _, cred := range creds {
		client, ok := h.clients[cred.ServerWorkload.Name]
		if !ok {
			continue
		}

		content, err := client.ReadResource(ctx, cred.AccessToken, req.URI)
		if err == nil && content != nil {
			if h.emitter != nil {
				h.emitter.EmitRequest(rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID)
				h.emitter.EmitResponse(rCtx.UserID, "MCP Client", cred.ServerWorkload.Name, cred.ServerWorkload.ID, 200)
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
