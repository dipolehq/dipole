/*
azure_key_vault.go retrieves secrets from Azure Key Vault. First obtains an
Azure access token via WIF, then calls the Key Vault REST API to get the
secret value.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type AzureKeyVaultHandler struct {
	wifHandler *AzureWIFHandler
	httpClient *http.Client
}

func NewAzureKeyVaultHandler(issuer *identity.IdentityIssuer) *AzureKeyVaultHandler {
	return &AzureKeyVaultHandler{
		wifHandler: NewAzureWIFHandler(issuer),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type azureKeyVaultConfig struct {
	VaultURL   string `json:"vault_url"`
	SecretName string `json:"secret_name"`
	TenantID   string `json:"tenant_id"`
	ClientID   string `json:"client_id"`
}

// issuerURL is the requesting tenant's identity host. It is forwarded to the
// Azure WIF handler, which uses it as the `iss` Entra verifies.
func (h *AzureKeyVaultHandler) Retrieve(ctx context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg azureKeyVaultConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse azure key vault config: %w", err)
	}

	// Get Azure token with Key Vault scope
	wifConfig, _ := json.Marshal(azureWIFConfig{
		TenantID: cfg.TenantID,
		ClientID: cfg.ClientID,
		Scope:    "https://vault.azure.net/.default",
	})

	azureCred, err := h.wifHandler.Retrieve(ctx, wifConfig, issuerURL, evidence, nil)
	if err != nil {
		return nil, fmt.Errorf("get azure token for key vault: %w", err)
	}

	secretValue, err := h.getSecret(ctx, cfg.VaultURL, cfg.SecretName, azureCred.Value)
	if err != nil {
		return nil, fmt.Errorf("get key vault secret: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: secretValue}, nil
}

func (h *AzureKeyVaultHandler) getSecret(ctx context.Context, vaultURL, secretName, token string) (string, error) {
	url := fmt.Sprintf("%s/secrets/%s?api-version=7.4", vaultURL, secretName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
		return "", fmt.Errorf("key vault returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Value, nil
}
