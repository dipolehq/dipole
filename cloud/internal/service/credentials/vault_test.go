package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestVault_AppRoleAuth_ReadSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/auth/approle/login":
			fmt.Fprint(w, `{"auth":{"client_token":"hvs.test-token-123"}}`)
		case "/v1/secret/data/myapp":
			if r.Header.Get("X-Vault-Token") != "hvs.test-token-123" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			fmt.Fprint(w, `{"data":{"data":{"password":"s3cr3t-value"}}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	handler := &VaultHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(vaultConfig{
		VaultURL:   server.URL,
		AuthMethod: "approle",
		AuthConfig: map[string]string{"role_id": "my-role", "secret_id": "my-secret"},
		SecretPath: "secret/data/myapp",
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Value != "s3cr3t-value" {
		t.Errorf("value = %q, want s3cr3t-value", cred.Value)
	}
}

func TestVault_TokenAuth_ReturnsToken(t *testing.T) {
	handler := NewVaultHandler()
	cfg, _ := json.Marshal(vaultConfig{
		VaultURL:   "http://unused",
		AuthMethod: "token",
		AuthConfig: map[string]string{"token": "hvs.direct-token"},
		SecretPath: "",
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Value != "hvs.direct-token" {
		t.Errorf("value = %q, want hvs.direct-token", cred.Value)
	}
}

func TestVault_AuthFailed_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"errors":["permission denied"]}`)
	}))
	defer server.Close()

	handler := &VaultHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(vaultConfig{
		VaultURL:   server.URL,
		AuthMethod: "approle",
		AuthConfig: map[string]string{"role_id": "bad", "secret_id": "bad"},
		SecretPath: "secret/data/x",
	})

	_, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error for auth failure")
	}
}

func TestVault_UnsupportedAuthMethod_Error(t *testing.T) {
	handler := NewVaultHandler()
	cfg, _ := json.Marshal(vaultConfig{
		VaultURL:   "http://x",
		AuthMethod: "unknown",
		AuthConfig: map[string]string{},
	})

	_, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error for unsupported auth method")
	}
}
