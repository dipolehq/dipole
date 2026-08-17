// events.go implements MethodHandler's security and workload event emission (access.* audit events and mcp.request/mcp.response logs) for the Identity Gateway.
package mcp_gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/google/uuid"
)

func (h *MethodHandler) emitRightSideAuth(ctx context.Context, rCtx *RequestContext, cred *ServerCredential, toolName string, allowed bool, denyReason string) uuid.UUID {
	contextID := uuid.New()

	if h.events == nil {
		return contextID
	}

	severity := "Info"
	if !allowed {
		severity = "Error"
	}

	serverName := ""
	var serverID *uuid.UUID
	var policyID *uuid.UUID
	policyName := ""
	credType := ""
	credName := ""
	var credProviderID *uuid.UUID
	serverHost := ""
	serverPort := 443
	rs := scope.DefaultResourceSetID

	if cred != nil {
		serverName = cred.ServerWorkload.Name
		id := cred.ServerWorkload.ID
		serverID = &id
		pid := cred.PolicyID
		policyID = &pid
		policyName = cred.PolicyName
		credType = cred.CredentialType
		credName = cred.CredentialName
		cpid := cred.CredentialProviderID
		credProviderID = &cpid
		serverHost = cred.ServerWorkload.Host
		serverPort = cred.ServerWorkload.Port
		rs = cred.ResourceSetID
	}

	clientWL := rCtx.ClientWorkloadName
	if clientWL == "" {
		clientWL = "Unknown"
	}

	envelope := events.SecurityEvent{
		TenantID:      rCtx.Scope.TenantID,
		ContextID:     contextID,
		SourceIP:      rCtx.ClientIP,
		Source:        "mcp_gateway",
		Severity:      "Info",
		ResourceSetID: &rs,
	}

	go func() {
		asyncCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()

		emit := func(event events.SecurityEvent) {
			if err := h.events.EmitSecurity(asyncCtx, event); err != nil {
				if h.logger != nil {
					h.logger.Error("failed to emit MCP access log", "event_type", event.EventType, "error", err, "server", serverName, "user", rCtx.UserID, "tool", toolName)
				}
			}
		}

		request := envelope
		request.EventType = "access.request"
		if !allowed {
			request.Severity = "Error"
		}
		request.Detail = events.RequestDetail{
			ServerHost: serverHost,
			ServerPort: serverPort,
		}
		emit(request)

		auth := envelope
		auth.EventType = "access.authorization"
		auth.Severity = severity
		auth.Detail = events.AuthorizationDetail{
			Allowed:              allowed,
			DenyReason:           denyReason,
			ClientWorkload:       clientWL,
			ServerWorkload:       serverName,
			ServerWorkloadID:     serverID,
			PolicyID:             policyID,
			PolicyName:           policyName,
			CredentialType:       credType,
			CredentialName:       credName,
			CredentialProviderID: credProviderID,
			ToolName:             toolName,
			UserID:               rCtx.UserID,
		}
		emit(auth)

		if allowed {
			credEvent := envelope
			credEvent.EventType = "access.credential"
			credEvent.Detail = events.CredentialDetail{
				Allowed:              allowed,
				DenyReason:           denyReason,
				ClientWorkload:       clientWL,
				ServerWorkload:       serverName,
				ServerWorkloadID:     serverID,
				PolicyID:             policyID,
				PolicyName:           policyName,
				CredentialType:       credType,
				CredentialName:       credName,
				CredentialProviderID: credProviderID,
			}
			emit(credEvent)
		}

		discovery := envelope
		discovery.EventType = "access.discovery"
		if !allowed {
			discovery.Severity = "Error"
		}
		var candidateServers []events.WorkloadRef
		if serverName != "" {
			ref := events.WorkloadRef{Name: serverName}
			if serverID != nil {
				ref.ID = *serverID
			}
			candidateServers = []events.WorkloadRef{ref}
		}
		discovery.Detail = events.DiscoveryDetail{
			CandidateServerWorkloads: candidateServers,
		}
		emit(discovery)
	}()

	return contextID
}

func (h *MethodHandler) emitWorkload(rCtx *RequestContext, server ServerInfo, resourceSetID uuid.UUID, mcpMethod string, toolName string, httpStatus int, rightAuthContextID uuid.UUID, responseHeaders http.Header) {
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
	serverID := server.ID

	var authChain []string
	if rCtx.AuthContextID != (uuid.UUID{}) {
		authChain = append(authChain, rCtx.AuthContextID.String())
	}
	if rightAuthContextID != (uuid.UUID{}) {
		authChain = append(authChain, rightAuthContextID.String())
	}

	rs := resourceSetID

	requestPayload := buildMCPPayload(mcpMethod, server, toolName, rCtx.UserID, rCtx.RequestID, rCtx.MCPSessionID, rCtx.ClientIP, rCtx.Headers, authChain, 0)
	h.events.EmitWorkload(events.WorkloadEvent{
		TenantID:         rCtx.Scope.TenantID,
		ConnectionID:     connID,
		EventType:        "mcp.request",
		Severity:         severity,
		ResourceSetID:    &rs,
		ClientWorkload:   "MCP Client",
		ServerWorkload:   server.Name,
		ServerWorkloadID: &serverID,
		Outcome:          outcome,
		HTTPMethod:       "POST",
		HTTPPath:         "/mcp",
		HTTPStatus:       httpStatus,
		UserID:           rCtx.UserID,
		Payload:          requestPayload,
	})

	responsePayload := buildMCPPayload(mcpMethod, server, toolName, rCtx.UserID, rCtx.RequestID, rCtx.MCPSessionID, rCtx.ClientIP, RedactHeaders(responseHeaders), authChain, httpStatus)
	h.events.EmitWorkload(events.WorkloadEvent{
		TenantID:         rCtx.Scope.TenantID,
		ConnectionID:     connID,
		EventType:        "mcp.response",
		Severity:         severity,
		ResourceSetID:    &rs,
		ClientWorkload:   "MCP Client",
		ServerWorkload:   server.Name,
		ServerWorkloadID: &serverID,
		Outcome:          outcome,
		HTTPMethod:       "POST",
		HTTPPath:         "/mcp",
		HTTPStatus:       httpStatus,
		UserID:           rCtx.UserID,
		Payload:          responsePayload,
	})
}

func buildMCPPayload(mcpMethod string, server ServerInfo, toolName string, userID string, requestID any, sessionID string, clientIP string, headers []map[string]string, authChain []string, httpResponseCode int) json.RawMessage {
	mcp := map[string]any{
		"httpMethod":         "POST",
		"httpVersion":        "HTTP/1.1",
		"uri":                "/mcp",
		"mcpMethod":          mcpMethod,
		"mcpServerName":      server.Name,
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
			"serverWorkloadHost": server.Host,
			"serverWorkloadPort": server.Port,
		}
	}

	if len(authChain) > 0 {
		payload["authorizationChain"] = authChain
	}
	data, _ := json.Marshal(payload)
	return data
}

var sensitiveHeaders = map[string]bool{
	"authorization": true,
	"cookie":        true,
	"set-cookie":    true,
	"x-api-key":     true,
}

func RedactHeaders(h http.Header) []map[string]string {
	if h == nil {
		return nil
	}
	headers := make([]map[string]string, 0, len(h))
	for name, values := range h {
		if sensitiveHeaders[strings.ToLower(name)] {
			headers = append(headers, map[string]string{"[REDACTED]": "[REDACTED]"})
			continue
		}
		for _, v := range values {
			headers = append(headers, map[string]string{name: v})
		}
	}
	return headers
}
