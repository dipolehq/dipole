/*
azure_wif.go exchanges our OIDC token for an Azure access token via Azure
Entra Workload Identity Federation. Issues an OIDC token with aud=client_id,
then exchanges it at Azure's token endpoint using the jwt-bearer assertion type.
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

type AzureWIFHandler struct {
	issuer     *identity.IdentityIssuer
	httpClient *http.Client
}

func NewAzureWIFHandler(issuer *identity.IdentityIssuer) *AzureWIFHandler {
	return &AzureWIFHandler{
		issuer:     issuer,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type azureWIFConfig struct {
	TenantID string `json:"tenant_id"`
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
}

// issuerURL is the requesting tenant's identity host. Azure Entra fetches OIDC
// discovery from the token's `iss` and rejects a mismatch with the federated
// credential the customer configured.
func (h *AzureWIFHandler) Retrieve(ctx context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg azureWIFConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse azure wif config: %w", err)
	}

	oidcToken, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL: issuerURL,
		Subject:   evidence.DipoleClientID,
		Audience:  cfg.ClientID,
		Lifetime:  5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("issue oidc token for azure: %w", err)
	}

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.TenantID)
	form := url.Values{
		"grant_type":            {"client_credentials"},
		"client_id":             {cfg.ClientID},
		"client_assertion_type": {"urn:ietf:params:oauth:client-assertion-type:jwt-bearer"},
		"client_assertion":      {oidcToken},
		"scope":                 {cfg.Scope},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create azure token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("azure token endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read azure response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("azure token endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse azure token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("azure response missing access_token")
	}

	return &domain.Credential{
		Type:      "bearer",
		Value:     tokenResp.AccessToken,
		ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
	}, nil
}
