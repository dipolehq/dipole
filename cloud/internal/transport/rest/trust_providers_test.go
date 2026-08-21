package rest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactTrustProviderConfig_RedactsSymmetricKeyIfPresent(t *testing.T) {
	config := json.RawMessage(`{"attestationMethod":"symmetric_key","symmetricKey":"super-secret-hmac-key"}`)
	redacted := redactTrustProviderConfig(config)

	if strings.Contains(string(redacted), "super-secret-hmac-key") {
		t.Errorf("expected symmetricKey to be redacted, got: %s", redacted)
	}
	if !strings.Contains(string(redacted), "REDACTED") {
		t.Errorf("expected REDACTED marker in output, got: %s", redacted)
	}
}

func TestRedactTrustProviderConfig_PreservesNonSecretFields(t *testing.T) {
	config := json.RawMessage(`{"attestationMethod":"oidc_discovery","oidcEndpoint":"https://accounts.google.com"}`)
	redacted := redactTrustProviderConfig(config)

	var parsed map[string]interface{}
	if err := json.Unmarshal(redacted, &parsed); err != nil {
		t.Fatalf("redacted config is not valid JSON: %v", err)
	}
	if parsed["attestationMethod"] != "oidc_discovery" {
		t.Errorf("expected attestationMethod to be preserved, got %v", parsed["attestationMethod"])
	}
	if parsed["oidcEndpoint"] != "https://accounts.google.com" {
		t.Errorf("expected oidcEndpoint to be preserved, got %v", parsed["oidcEndpoint"])
	}
}

func TestRedactTrustProviderConfig_EmptyOrNilConfig_ReturnsEmptyObject(t *testing.T) {
	if got := string(redactTrustProviderConfig(nil)); got != "{}" {
		t.Errorf("expected {} for nil config, got %s", got)
	}
	if got := string(redactTrustProviderConfig(json.RawMessage(``))); got != "{}" {
		t.Errorf("expected {} for empty config, got %s", got)
	}
}

func TestRedactTrustProviderConfig_InvalidJSON_ReturnsEmptyObject(t *testing.T) {
	if got := string(redactTrustProviderConfig(json.RawMessage(`not json`))); got != "{}" {
		t.Errorf("expected {} for invalid JSON, got %s", got)
	}
}
