/*
mcp_user_token.go handles MCP User-Based Access Token credential retrieval.
Per-user OAuth tokens for MCP Identity Gateway — each user gets their own
credential. Only usable via MCP Identity Gateway (Job 11), not the normal
Agent Proxy path which cannot trigger browser-based OAuth consent flows.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type MCPUserTokenHandler struct{}

func NewMCPUserTokenHandler() *MCPUserTokenHandler {
	return &MCPUserTokenHandler{}
}

type mcpUserTokenConfig struct {
	MCPServerURL     string `json:"mcp_server_url"`
	AuthorizationURL string `json:"authorization_url"`
	TokenURL         string `json:"token_url"`
	ClientID         string `json:"client_id"`
	ClientSecret     string `json:"client_secret"`
	Scopes           string `json:"scopes"`
	PKCERequired     bool   `json:"pkce_required"`
}

func (h *MCPUserTokenHandler) Retrieve(_ context.Context, configJSON []byte, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg mcpUserTokenConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse mcp user token config: %w", err)
	}

	// MCP User-Based tokens require the MCP Identity Gateway flow (Job 11).
	// The Agent Proxy cannot trigger browser-based OAuth consent.
	// The MCP Gateway handles per-user token acquisition and storage.
	return nil, fmt.Errorf("MCP User-Based tokens require MCP Identity Gateway")
}
