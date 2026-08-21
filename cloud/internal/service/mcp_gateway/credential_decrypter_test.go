package mcp_gateway

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestAESCredentialDecrypter_APIKey_ReturnsBearerHeader(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	cfg, _ := json.Marshal(map[string]string{"apiKey": "my-secret-key"})
	encrypted, err := testEncrypt(cfg, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	d := NewAESCredentialDecrypter(encKey)

	got, err := d.DecryptCredential(encrypted, "api_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Bearer my-secret-key" {
		t.Errorf("expected 'Bearer my-secret-key', got %q", got)
	}
}

func TestAESCredentialDecrypter_UsernamePassword_ReturnsBasicHeader(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 1)
	}

	cfg, _ := json.Marshal(map[string]string{"username": "alice", "password": "s3cr3t"})
	encrypted, err := testEncrypt(cfg, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	d := NewAESCredentialDecrypter(encKey)

	got, err := d.DecryptCredential(encrypted, "username_password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantEncoded := base64.StdEncoding.EncodeToString([]byte("alice:s3cr3t"))
	want := "Basic " + wantEncoded
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestAESCredentialDecrypter_UsernamePassword_EmptyUsername_StillEncodesColon(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 2)
	}

	cfg, _ := json.Marshal(map[string]string{"password": "token-only-password"})
	encrypted, err := testEncrypt(cfg, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	d := NewAESCredentialDecrypter(encKey)

	got, err := d.DecryptCredential(encrypted, "username_password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantEncoded := base64.StdEncoding.EncodeToString([]byte(":token-only-password"))
	want := "Basic " + wantEncoded
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestAESCredentialDecrypter_UsernamePassword_EmptyPassword_Error(t *testing.T) {
	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i + 3)
	}

	cfg, _ := json.Marshal(map[string]string{"username": "alice", "password": ""})
	encrypted, err := testEncrypt(cfg, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	d := NewAESCredentialDecrypter(encKey)

	_, err = d.DecryptCredential(encrypted, "username_password")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestAESCredentialDecrypter_EmptyConfig_Error(t *testing.T) {
	d := NewAESCredentialDecrypter(make([]byte, 32))

	_, err := d.DecryptCredential(nil, "api_key")
	if err == nil {
		t.Fatal("expected error for empty credential config")
	}
}

func TestAESCredentialDecrypter_UnsupportedType_Error(t *testing.T) {
	encKey := make([]byte, 32)
	cfg, _ := json.Marshal(map[string]string{"apiKey": "x"})
	encrypted, err := testEncrypt(cfg, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	d := NewAESCredentialDecrypter(encKey)

	_, err = d.DecryptCredential(encrypted, "oauth_authorization_code")
	if err == nil {
		t.Fatal("expected error for a credential type this decrypter does not handle")
	}
}
