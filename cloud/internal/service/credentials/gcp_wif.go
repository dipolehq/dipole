// gcp_wif.go exchanges our OIDC token for a GCP access token via Google Workload Identity Federation, then impersonates a service account.
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

type GCPWIFHandler struct {
	issuer     *identity.IdentityIssuer
	httpClient *http.Client
}

func NewGCPWIFHandler(issuer *identity.IdentityIssuer) *GCPWIFHandler {
	return &GCPWIFHandler{
		issuer:     issuer,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type gcpWIFConfig struct {
	ProjectNumber       string `json:"project_number"`
	PoolID              string `json:"pool_id"`
	ProviderID          string `json:"provider_id"`
	ServiceAccountEmail string `json:"service_account_email"`
	Scopes              string `json:"scopes"`
}

func (h *GCPWIFHandler) Retrieve(ctx context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg gcpWIFConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse gcp wif config: %w", err)
	}

	audience := fmt.Sprintf("//iam.googleapis.com/projects/%s/locations/global/workloadIdentityPools/%s/providers/%s",
		cfg.ProjectNumber, cfg.PoolID, cfg.ProviderID)

	oidcToken, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL: issuerURL,
		Subject:   evidence.DipoleClientID,
		Audience:  audience,
		Lifetime:  5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("issue oidc token for gcp: %w", err)
	}

	federatedToken, err := h.exchangeToken(ctx, oidcToken, audience)
	if err != nil {
		return nil, fmt.Errorf("gcp sts exchange: %w", err)
	}

	accessToken, err := h.impersonateServiceAccount(ctx, federatedToken, cfg.ServiceAccountEmail, cfg.Scopes)
	if err != nil {
		return nil, fmt.Errorf("gcp sa impersonation: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: accessToken}, nil
}

func (h *GCPWIFHandler) exchangeToken(ctx context.Context, oidcToken, audience string) (string, error) {
	form := url.Values{
		"grant_type":         {"urn:ietf:params:oauth:grant-type:token-exchange"},
		"subject_token":      {oidcToken},
		"subject_token_type": {"urn:ietf:params:oauth:token-type:jwt"},
		"audience":           {audience},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sts.googleapis.com/v1/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google sts returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

func (h *GCPWIFHandler) impersonateServiceAccount(ctx context.Context, federatedToken, saEmail, scopes string) (string, error) {
	impURL := fmt.Sprintf("https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken", saEmail)

	scopeList := strings.Split(scopes, " ")
	if scopes == "" {
		scopeList = []string{"https://www.googleapis.com/auth/cloud-platform"}
	}

	reqBody, _ := json.Marshal(map[string]interface{}{
		"scope": scopeList,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, impURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+federatedToken)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gcp impersonation returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}
