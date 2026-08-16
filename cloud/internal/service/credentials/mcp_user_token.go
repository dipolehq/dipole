// mcp_user_token.go handles MCP User-Based Access Token credential retrieval, which is only usable via the MCP Identity Gateway, not this broker.
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

	return nil, fmt.Errorf("MCP User-Based tokens require MCP Identity Gateway")
}
