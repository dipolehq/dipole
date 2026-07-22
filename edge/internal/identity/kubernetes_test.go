package identity

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestDecodeJWTPayload_ValidToken(t *testing.T) {
	// Build a fake JWT with a known payload
	payload := map[string]interface{}{
		"iss": "https://kubernetes.default.svc.cluster.local",
		"sub": "system:serviceaccount:production:myapp-sa",
		"kubernetes.io": map[string]interface{}{
			"namespace": "production",
			"serviceaccount": map[string]interface{}{
				"name": "myapp-sa",
				"uid":  "abc-123-def",
			},
		},
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// JWT = header.payload.signature
	token := "eyJhbGciOiJSUzI1NiJ9." + payloadB64 + ".fake-signature"

	claims, err := decodeJWTPayload(token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if claims["iss"] != "https://kubernetes.default.svc.cluster.local" {
		t.Errorf("expected iss claim, got %v", claims["iss"])
	}
	if claims["sub"] != "system:serviceaccount:production:myapp-sa" {
		t.Errorf("expected sub claim, got %v", claims["sub"])
	}
}

func TestDecodeJWTPayload_InvalidToken(t *testing.T) {
	_, err := decodeJWTPayload("not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid JWT, got nil")
	}
}

func TestDecodeJWTPayload_InvalidBase64(t *testing.T) {
	// Valid structure but invalid base64 in the payload segment
	_, err := decodeJWTPayload("header.!!!invalid!!!.signature")
	if err == nil {
		t.Fatal("expected error for invalid base64, got nil")
	}
}

func TestDecodeJWTPayload_InvalidJSON(t *testing.T) {
	// Valid base64 but not valid JSON
	notJSON := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	_, err := decodeJWTPayload("header." + notJSON + ".signature")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestExtractServiceAccount_ValidClaims(t *testing.T) {
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"serviceaccount": map[string]interface{}{
				"name": "myapp-sa",
				"uid":  "abc-123-def",
			},
		},
	}

	name, uid := extractServiceAccount(claims)
	if name != "myapp-sa" {
		t.Errorf("expected name 'myapp-sa', got '%s'", name)
	}
	if uid != "abc-123-def" {
		t.Errorf("expected uid 'abc-123-def', got '%s'", uid)
	}
}

func TestExtractServiceAccount_MissingK8sClaims(t *testing.T) {
	claims := map[string]interface{}{
		"iss": "something",
	}

	name, uid := extractServiceAccount(claims)
	if name != "" || uid != "" {
		t.Errorf("expected empty strings, got name='%s' uid='%s'", name, uid)
	}
}

func TestExtractServiceAccount_MissingServiceAccount(t *testing.T) {
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"namespace": "production",
		},
	}

	name, uid := extractServiceAccount(claims)
	if name != "" || uid != "" {
		t.Errorf("expected empty strings, got name='%s' uid='%s'", name, uid)
	}
}

func TestExtractServiceAccount_PartialFields(t *testing.T) {
	// Only name present, no uid
	claims := map[string]interface{}{
		"kubernetes.io": map[string]interface{}{
			"serviceaccount": map[string]interface{}{
				"name": "only-name",
			},
		},
	}

	name, uid := extractServiceAccount(claims)
	if name != "only-name" {
		t.Errorf("expected name 'only-name', got '%s'", name)
	}
	if uid != "" {
		t.Errorf("expected empty uid, got '%s'", uid)
	}
}
