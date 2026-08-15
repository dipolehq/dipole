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
	"net/http"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/google/uuid"
)

const toolPrefixSeparator = "__"

var ErrMethodNotFound = errors.New("method not found")

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
	ListTools(ctx context.Context, token string) ([]MCPTool, http.Header, error)
	CallTool(ctx context.Context, token string, toolName string, args json.RawMessage) (any, http.Header, error)
	ListResources(ctx context.Context, token string) ([]MCPResource, http.Header, error)
	ReadResource(ctx context.Context, token string, uri string) (*MCPResourceContent, http.Header, error)
}

type RequestContext struct {
	UserID        string
	Claims        map[string]any
	Scope         TenantScope
	RequestID     any
	AuthContextID uuid.UUID
	MCPSessionID  string
	ClientIP      string
	Headers       []map[string]string
}

type MethodHandler struct {
	rightAuth *RightSideAuth
	clients   MCPClientFactory
	events    *events.EventBus
	logger    *slog.Logger
}

func NewMethodHandler(rightAuth *RightSideAuth, clients MCPClientFactory, eventBus *events.EventBus, logger *slog.Logger) *MethodHandler {
	return &MethodHandler{
		rightAuth: rightAuth,
		clients:   clients,
		events:    eventBus,
		logger:    logger,
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

		tools, _, err := client.ListTools(ctx, cred.AccessToken)
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
		h.emitRightSideAuth(ctx, rCtx, serverName, nil, toolName, false, err.Error())
		return nil, err
	}

	client := h.clients.ClientFor(cred.ServerWorkload)

	result, respHeaders, err := client.CallTool(ctx, cred.AccessToken, toolName, req.Arguments)
	if err != nil {
		rightContextID := h.emitRightSideAuth(ctx, rCtx, cred.ServerWorkload.Name, &cred.ServerWorkload.ID, toolName, false, err.Error())
		h.emitWorkload(rCtx, cred.ServerWorkload.Name, cred.ServerWorkload.ID, "tools/call", toolName, 500, rightContextID, nil)
		return nil, err
	}

	rightContextID := h.emitRightSideAuth(ctx, rCtx, cred.ServerWorkload.Name, &cred.ServerWorkload.ID, toolName, true, "")
	h.emitWorkload(rCtx, cred.ServerWorkload.Name, cred.ServerWorkload.ID, "tools/call", toolName, 200, rightContextID, respHeaders)

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

		resources, _, err := client.ListResources(ctx, cred.AccessToken)
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

		content, respHeaders, err := client.ReadResource(ctx, cred.AccessToken, req.URI)
		if err == nil && content != nil {
			h.emitWorkload(rCtx, cred.ServerWorkload.Name, cred.ServerWorkload.ID, "resources/read", "", 200, uuid.Nil, respHeaders)
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

func (h *MethodHandler) emitRightSideAuth(ctx context.Context, rCtx *RequestContext, serverName string, serverID *uuid.UUID, toolName string, allowed bool, denyReason string) uuid.UUID {
	contextID := uuid.New()

	if h.events == nil {
		return contextID
	}

	severity := "Info"
	if !allowed {
		severity = "Error"
	}

	policyName := "Gateway → " + serverName

	base := events.SecurityEvent{
		TenantID:         rCtx.Scope.TenantID,
		ContextID:        contextID,
		SourceIP:         rCtx.UserID,
		Source:           "mcp_gateway",
		Severity:         "Info",
		Allowed:          allowed,
		DenyReason:       denyReason,
		UserID:           rCtx.UserID,
		ClientWorkload:   "MCP Identity Gateway",
		ServerWorkload:   serverName,
		ServerWorkloadID: serverID,
		ServerHost:       serverName,
		ServerPort:       443,
		PolicyName:       policyName,
		ToolName:         toolName,
	}

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		request := base
		request.EventType = "access.request"
		if !allowed {
			request.Severity = "Error"
		}
		h.events.EmitSecurity(asyncCtx, request)

		auth := base
		auth.EventType = "access.authorization"
		auth.Severity = severity
		h.events.EmitSecurity(asyncCtx, auth)

		if allowed {
			cred := base
			cred.EventType = "access.credential"
			h.events.EmitSecurity(asyncCtx, cred)
		}

		discovery := base
		discovery.EventType = "access.discovery"
		if !allowed {
			discovery.Severity = "Error"
		}
		discovery.CandidateServerWorkloads = []events.WorkloadRef{{Name: serverName}}
		if serverID != nil {
			discovery.CandidateServerWorkloads[0].ID = *serverID
		}
		if err := h.events.EmitSecurity(asyncCtx, discovery); err != nil {
			if h.logger != nil {
				h.logger.Error("failed to emit MCP access log", "error", err, "server", serverName, "user", rCtx.UserID, "tool", toolName)
			}
		}
	}()

	return contextID
}

func (h *MethodHandler) emitWorkload(rCtx *RequestContext, serverName string, serverID uuid.UUID, mcpMethod string, toolName string, httpStatus int, rightAuthContextID uuid.UUID, responseHeaders http.Header) {
	if h.events == nil {
		return
	}

	outcome := "Modified"
	severity := "Info"
	if httpStatus >= 400 {
		outcome = "Error"
		severity = "Error"
	}

	connID := uuid.New()

	var authChain []string
	if rCtx.AuthContextID != (uuid.UUID{}) {
		authChain = append(authChain, rCtx.AuthContextID.String())
	}
	if rightAuthContextID != (uuid.UUID{}) {
		authChain = append(authChain, rightAuthContextID.String())
	}

	requestPayload := buildMCPPayload(mcpMethod, serverName, toolName, rCtx.UserID, rCtx.RequestID, rCtx.MCPSessionID, rCtx.ClientIP, rCtx.Headers, authChain, 0)
	h.events.EmitWorkload(events.WorkloadEvent{
		TenantID:         rCtx.Scope.TenantID,
		ConnectionID:     connID,
		EventType:        "mcp.request",
		Severity:         severity,
		ClientWorkload:   "MCP Client",
		ServerWorkload:   serverName,
		ServerWorkloadID: &serverID,
		Outcome:          outcome,
		HTTPMethod:       "POST",
		HTTPPath:         "/mcp",
		HTTPStatus:       httpStatus,
		UserID:           rCtx.UserID,
		Payload:          requestPayload,
	})

	responsePayload := buildMCPPayload(mcpMethod, serverName, toolName, rCtx.UserID, rCtx.RequestID, rCtx.MCPSessionID, rCtx.ClientIP, redactHTTPHeaders(responseHeaders), authChain, httpStatus)
	h.events.EmitWorkload(events.WorkloadEvent{
		TenantID:         rCtx.Scope.TenantID,
		ConnectionID:     connID,
		EventType:        "mcp.response",
		Severity:         severity,
		ClientWorkload:   "MCP Client",
		ServerWorkload:   serverName,
		ServerWorkloadID: &serverID,
		Outcome:          outcome,
		HTTPMethod:       "POST",
		HTTPPath:         "/mcp",
		HTTPStatus:       httpStatus,
		UserID:           rCtx.UserID,
		Payload:          responsePayload,
	})
}

func buildMCPPayload(mcpMethod string, serverName string, toolName string, userID string, requestID any, sessionID string, clientIP string, headers []map[string]string, authChain []string, httpResponseCode int) json.RawMessage {
	mcp := map[string]any{
		"httpMethod":         "POST",
		"httpVersion":        "HTTP/1.1",
		"uri":                "/mcp",
		"mcpMethod":          mcpMethod,
		"mcpServerName":      serverName,
		"mcpProtocolVersion": "2024-11-05",
		"userId":             userID,
	}
	if toolName != "" {
		mcp["mcpToolName"] = toolName
	}
	if requestID != nil {
		mcp["mcpRequestId"] = requestID
	}
	if sessionID != "" {
		mcp["mcpSessionId"] = sessionID
	}
	if len(headers) > 0 {
		mcp["headers"] = headers
	}
	if httpResponseCode > 0 {
		mcp["httpResponseCode"] = httpResponseCode
	}

	payload := map[string]any{"mcp": mcp}

	if clientIP != "" {
		payload["network"] = map[string]any{
			"clientWorkloadIP":   clientIP,
			"transportProtocol":  "TCP",
			"serverWorkloadHost": serverName,
			"serverWorkloadPort": 443,
		}
	}

	if len(authChain) > 0 {
		payload["authorizationChain"] = authChain
	}
	data, _ := json.Marshal(payload)
	return data
}

var mcpSensitiveHeaders = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"set-cookie":    true,
	"x-api-key":     true,
}

func redactHTTPHeaders(h http.Header) []map[string]string {
	if h == nil {
		return nil
	}
	headers := make([]map[string]string, 0, len(h))
	for name, values := range h {
		if mcpSensitiveHeaders[strings.ToLower(name)] {
			headers = append(headers, map[string]string{"[REDACTED]": "[REDACTED]"})
			continue
		}
		for _, v := range values {
			headers = append(headers, map[string]string{name: v})
		}
	}
	return headers
}
