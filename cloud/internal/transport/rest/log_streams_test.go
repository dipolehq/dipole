package rest

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactLogStreamConfig_RedactsAllSensitiveKeysAcrossDestinationTypes(t *testing.T) {
	cases := []struct {
		name   string
		config string
		secret string
	}{
		{"splunk hec token", `{"hec_url":"https://x","hec_token":"super-secret","index":"main"}`, "super-secret"},
		{"crowdstrike client secret", `{"api_url":"https://x","client_id":"id","client_secret":"super-secret"}`, "super-secret"},
		{"s3 access key id", `{"bucket_name":"b","region":"us-east-1","access_key_id":"AKIA-super-secret","secret_access_key":"s"}`, "AKIA-super-secret"},
		{"s3 secret access key", `{"bucket_name":"b","region":"us-east-1","access_key_id":"a","secret_access_key":"super-secret"}`, "super-secret"},
		{"gcs service account json", `{"bucket_name":"b","service_account_json":"super-secret"}`, "super-secret"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			redacted := redactLogStreamConfig(json.RawMessage(tc.config))
			if strings.Contains(string(redacted), tc.secret) {
				t.Errorf("expected secret %q to be redacted, got: %s", tc.secret, redacted)
			}
			if !strings.Contains(string(redacted), "REDACTED") {
				t.Errorf("expected REDACTED marker in output, got: %s", redacted)
			}
		})
	}
}

func TestRedactLogStreamConfig_PreservesNonSensitiveFields(t *testing.T) {
	config := json.RawMessage(`{"bucket_name":"my-bucket","region":"us-east-1","access_key_id":"a","secret_access_key":"s"}`)
	redacted := redactLogStreamConfig(config)

	var parsed map[string]interface{}
	if err := json.Unmarshal(redacted, &parsed); err != nil {
		t.Fatalf("redacted config is not valid JSON: %v", err)
	}
	if parsed["bucket_name"] != "my-bucket" {
		t.Errorf("expected bucket_name to be preserved, got %v", parsed["bucket_name"])
	}
	if parsed["region"] != "us-east-1" {
		t.Errorf("expected region to be preserved, got %v", parsed["region"])
	}
	if parsed["secret_access_key"] == "s" {
		t.Error("expected secret_access_key to be redacted, not passed through")
	}
}

func TestRedactLogStreamConfig_EmptyOrNilConfig_ReturnsEmptyObject(t *testing.T) {
	if got := string(redactLogStreamConfig(nil)); got != "{}" {
		t.Errorf("expected {} for nil config, got %s", got)
	}
	if got := string(redactLogStreamConfig(json.RawMessage(``))); got != "{}" {
		t.Errorf("expected {} for empty config, got %s", got)
	}
}

func TestRedactLogStreamConfig_InvalidJSON_ReturnsEmptyObject(t *testing.T) {
	if got := string(redactLogStreamConfig(json.RawMessage(`not json`))); got != "{}" {
		t.Errorf("expected {} for invalid JSON, got %s", got)
	}
}
