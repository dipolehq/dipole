// vault.go retrieves secrets from HashiCorp Vault, supporting AppRole and direct-token authentication.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type VaultHandler struct {
	httpClient *http.Client
}

func NewVaultHandler() *VaultHandler {
	return &VaultHandler{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type vaultConfig struct {
	VaultURL   string            `json:"vault_url"`
	AuthMethod string            `json:"auth_method"`
	AuthConfig map[string]string `json:"auth_config"`
	SecretPath string            `json:"secret_path"`
}

func (h *VaultHandler) Retrieve(ctx context.Context, configJSON []byte, _ *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg vaultConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse vault config: %w", err)
	}

	clientToken, err := h.authenticate(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("vault auth: %w", err)
	}

	if cfg.SecretPath == "" {
		return &domain.Credential{Type: "bearer", Value: clientToken}, nil
	}

	secretValue, err := h.readSecret(ctx, cfg.VaultURL, cfg.SecretPath, clientToken)
	if err != nil {
		return nil, fmt.Errorf("vault read secret: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: secretValue}, nil
}

func (h *VaultHandler) authenticate(ctx context.Context, cfg vaultConfig) (string, error) {
	switch cfg.AuthMethod {
	case "token":
		token := cfg.AuthConfig["token"]
		if token == "" {
			return "", fmt.Errorf("vault token not configured")
		}
		return token, nil

	case "approle":
		return h.authAppRole(ctx, cfg.VaultURL, cfg.AuthConfig["role_id"], cfg.AuthConfig["secret_id"])

	default:
		return "", fmt.Errorf("unsupported vault auth method: %s", cfg.AuthMethod)
	}
}

func (h *VaultHandler) authAppRole(ctx context.Context, vaultURL, roleID, secretID string) (string, error) {
	loginURL := vaultURL + "/v1/auth/approle/login"
	body, _ := json.Marshal(map[string]string{
		"role_id":   roleID,
		"secret_id": secretID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("vault approle login returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	if result.Auth.ClientToken == "" {
		return "", fmt.Errorf("vault response missing client_token")
	}

	return result.Auth.ClientToken, nil
}

func (h *VaultHandler) readSecret(ctx context.Context, vaultURL, secretPath, token string) (string, error) {
	url := vaultURL + "/v1/" + secretPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Vault-Token", token)

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
		return "", fmt.Errorf("vault read returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Data map[string]interface{} `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	for _, v := range result.Data.Data {
		return fmt.Sprint(v), nil
	}

	return string(body), nil
}
