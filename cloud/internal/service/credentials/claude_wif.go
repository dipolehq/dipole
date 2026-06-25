/*
claude_wif.go exchanges our OIDC token for a Claude API access token via
Anthropic's Workload Identity Federation endpoint. Issues an OIDC token
and exchanges it with Claude's token service.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type ClaudeWIFHandler struct {
	issuer     *identity.IdentityIssuer
	httpClient *http.Client
}

func NewClaudeWIFHandler(issuer *identity.IdentityIssuer) *ClaudeWIFHandler {
	return &ClaudeWIFHandler{
		issuer:     issuer,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type claudeWIFConfig struct {
	OrganizationID   string `json:"organization_id"`
	ServiceAccountID string `json:"service_account_id"`
	WorkspaceID      string `json:"workspace_id"`
	Scope            string `json:"scope"`
}

func (h *ClaudeWIFHandler) Retrieve(ctx context.Context, configJSON []byte, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg claudeWIFConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse claude wif config: %w", err)
	}

	oidcToken, err := h.issuer.IssueToken(identity.IssuanceRequest{
		Subject:  evidence.DipoleClientID,
		Audience: "api.anthropic.com",
		Lifetime: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("issue oidc token for claude: %w", err)
	}

	tokenURL := "https://api.anthropic.com/v1/oauth/token"
	form := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {oidcToken},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
		"organization_id":    {cfg.OrganizationID},
		"service_account_id": {cfg.ServiceAccountID},
	}
	if cfg.WorkspaceID != "" {
		form.Set("workspace_id", cfg.WorkspaceID)
	}
	if cfg.Scope != "" {
		form.Set("scope", cfg.Scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create claude token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude token endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read claude response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("claude token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse claude token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("claude response missing access_token")
	}

	return &domain.Credential{
		Type:      "bearer",
		Value:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
	}, nil
}
