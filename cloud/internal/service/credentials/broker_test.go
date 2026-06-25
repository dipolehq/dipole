package credentials

import (
	"context"
	"testing"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

var testKey = []byte("01234567890123456789012345678901") // 32 bytes

func testIssuer(t *testing.T) *identity.IdentityIssuer {
	t.Helper()
	issuer, err := identity.NewIdentityIssuer("test", "ap-south-1")
	if err != nil {
		t.Fatal(err)
	}
	return issuer
}

func encrypt(t *testing.T, plaintext string) []byte {
	t.Helper()
	ct, err := postgres.Encrypt([]byte(plaintext), testKey)
	if err != nil {
		t.Fatal(err)
	}
	return ct
}

func TestRetrieve_APIKey(t *testing.T) {
	broker := NewCredentialBroker(testKey, testIssuer(t))
	provider := &domain.CredentialProvider{
		Type:            "api_key",
		ConfigEncrypted: encrypt(t, `{"api_key":"sk_live_test123"}`),
	}

	cred, err := broker.Retrieve(context.Background(), provider, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Type != "bearer" {
		t.Errorf("type = %q, want %q", cred.Type, "bearer")
	}
	if cred.Value != "sk_live_test123" {
		t.Errorf("value = %q, want %q", cred.Value, "sk_live_test123")
	}
}

func TestRetrieve_UsernamePassword(t *testing.T) {
	broker := NewCredentialBroker(testKey, testIssuer(t))
	provider := &domain.CredentialProvider{
		Type:            "username_password",
		ConfigEncrypted: encrypt(t, `{"username":"admin","password":"s3cr3t"}`),
	}

	cred, err := broker.Retrieve(context.Background(), provider, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Type != "basic" {
		t.Errorf("type = %q, want %q", cred.Type, "basic")
	}
	if cred.Username != "admin" {
		t.Errorf("username = %q, want %q", cred.Username, "admin")
	}
	if cred.Password != "s3cr3t" {
		t.Errorf("password = %q, want %q", cred.Password, "s3cr3t")
	}
}

func TestRetrieve_WrongKey_ReturnsError(t *testing.T) {
	broker := NewCredentialBroker([]byte("wrong-key-wrong-key-wrong-key!!"), testIssuer(t))
	provider := &domain.CredentialProvider{
		Type:            "api_key",
		ConfigEncrypted: encrypt(t, `{"api_key":"x"}`),
	}

	_, err := broker.Retrieve(context.Background(), provider, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error with wrong decryption key")
	}
}

func TestRetrieve_UnsupportedType_ReturnsError(t *testing.T) {
	broker := NewCredentialBroker(testKey, testIssuer(t))
	provider := &domain.CredentialProvider{
		Type:            "nonexistent_type",
		ConfigEncrypted: encrypt(t, `{}`),
	}

	_, err := broker.Retrieve(context.Background(), provider, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}
