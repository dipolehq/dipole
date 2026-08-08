package credentials

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	"github.com/google/uuid"
)

var testKey = []byte("01234567890123456789012345678901") // 32 bytes

func testIssuer(t *testing.T) *identity.IdentityIssuer {
	t.Helper()
	issuer, err := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}
	return issuer
}

// stubTenantLookup lets the broker resolve a tenant subdomain, which it needs to
// build the per-tenant token issuer URL.
type stubTenantLookup struct {
	tenant *domain.Tenant
	err    error
}

func (s *stubTenantLookup) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.tenant == nil {
		return nil, nil
	}
	return s.tenant, nil
}

func testBroker(t *testing.T) *CredentialBrokerImpl {
	t.Helper()
	return NewCredentialBroker(
		testKey,
		testIssuer(t),
		&stubTenantLookup{tenant: &domain.Tenant{ID: testTenantID, Subdomain: "647251"}},
		platform.NewHosts("dipolehq.com"),
	)
}

var testTenantID = uuid.New()

func encrypt(t *testing.T, plaintext string) []byte {
	t.Helper()
	ct, err := postgres.Encrypt([]byte(plaintext), testKey)
	if err != nil {
		t.Fatal(err)
	}
	return ct
}

func TestRetrieve_APIKey(t *testing.T) {
	broker := testBroker(t)
	provider := &domain.CredentialProvider{
		Type:            "api_key",
		ConfigEncrypted: encrypt(t, `{"api_key":"sk_live_test123"}`),
	}

	cred, err := broker.Retrieve(context.Background(), testTenantID, provider, &domain.ClientEvidence{}, nil)
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
	broker := testBroker(t)
	provider := &domain.CredentialProvider{
		Type:            "username_password",
		ConfigEncrypted: encrypt(t, `{"username":"admin","password":"s3cr3t"}`),
	}

	cred, err := broker.Retrieve(context.Background(), testTenantID, provider, &domain.ClientEvidence{}, nil)
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
	broker := NewCredentialBroker(
		[]byte("wrong-key-wrong-key-wrong-key!!"),
		testIssuer(t),
		&stubTenantLookup{tenant: &domain.Tenant{ID: testTenantID, Subdomain: "647251"}},
		platform.NewHosts("dipolehq.com"),
	)
	provider := &domain.CredentialProvider{
		Type:            "api_key",
		ConfigEncrypted: encrypt(t, `{"api_key":"x"}`),
	}

	_, err := broker.Retrieve(context.Background(), testTenantID, provider, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error with wrong decryption key")
	}
}

func TestRetrieve_UnsupportedType_ReturnsError(t *testing.T) {
	broker := testBroker(t)
	provider := &domain.CredentialProvider{
		Type:            "nonexistent_type",
		ConfigEncrypted: encrypt(t, `{}`),
	}

	_, err := broker.Retrieve(context.Background(), testTenantID, provider, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

// decodeJWTClaim pulls a single claim out of a JWT payload without verifying the
// signature — enough to assert which issuer was stamped on the token.
func decodeJWTClaim(t *testing.T, token, claim string) string {
	t.Helper()

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatalf("parse JWT payload: %v", err)
	}
	s, _ := claims[claim].(string)
	return s
}

// The whole point of threading tenantID into Retrieve: a token we sign must
// carry the REQUESTING tenant's identity host as `iss`. AWS STS / Azure Entra /
// GCP WIF fetch OIDC discovery from `iss` and reject a mismatch, so a shared or
// stale issuer silently breaks federation for every tenant but one.
func TestRetrieve_SelfSignedToken_UsesRequestingTenantIssuer(t *testing.T) {
	tenantA := uuid.New()
	tenantB := uuid.New()

	lookup := &multiTenantLookup{tenants: map[uuid.UUID]*domain.Tenant{
		tenantA: {ID: tenantA, Subdomain: "111111"},
		tenantB: {ID: tenantB, Subdomain: "222222"},
	}}
	broker := NewCredentialBroker(testKey, testIssuer(t), lookup, platform.NewHosts("dipolehq.com"))

	newProvider := func() *domain.CredentialProvider {
		return &domain.CredentialProvider{
			// Distinct IDs so the credential cache can't serve one tenant's
			// token to the other.
			ID:              uuid.New(),
			Type:            "oidc_id_token",
			ConfigEncrypted: encrypt(t, `{"subject":"workload","audience":"https://api.example.com"}`),
		}
	}

	credA, err := broker.Retrieve(context.Background(), tenantA, newProvider(), &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatalf("tenant A retrieve: %v", err)
	}
	credB, err := broker.Retrieve(context.Background(), tenantB, newProvider(), &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatalf("tenant B retrieve: %v", err)
	}

	issA := decodeJWTClaim(t, credA.Value, "iss")
	issB := decodeJWTClaim(t, credB.Value, "iss")

	if want := "https://111111-id.dipolehq.com"; issA != want {
		t.Errorf("tenant A iss = %q, want %q", issA, want)
	}
	if want := "https://222222-id.dipolehq.com"; issB != want {
		t.Errorf("tenant B iss = %q, want %q", issB, want)
	}
	if issA == issB {
		t.Error("both tenants received the same issuer — federation would break for one of them")
	}
}

// A credential type that never embeds our issuer must not require a tenant
// lookup, so a lookup failure can't break unrelated credential types.
func TestRetrieve_StoredSecret_DoesNotRequireTenantLookup(t *testing.T) {
	broker := NewCredentialBroker(
		testKey,
		testIssuer(t),
		&stubTenantLookup{err: errors.New("tenant lookup unavailable")},
		platform.NewHosts("dipolehq.com"),
	)
	provider := &domain.CredentialProvider{
		Type:            "api_key",
		ConfigEncrypted: encrypt(t, `{"api_key":"sk_live_test123"}`),
	}

	cred, err := broker.Retrieve(context.Background(), uuid.New(), provider, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatalf("api_key retrieve should not depend on tenant resolution: %v", err)
	}
	if cred.Value != "sk_live_test123" {
		t.Errorf("value = %q", cred.Value)
	}
}

// Fail closed: if we can't determine the tenant's issuer, refuse rather than
// signing a token with the wrong `iss` that the cloud provider will reject
// opaquely later.
func TestRetrieve_SelfSignedToken_UnknownTenantFailsClosed(t *testing.T) {
	broker := NewCredentialBroker(
		testKey,
		testIssuer(t),
		&stubTenantLookup{tenant: nil}, // resolves to "not found"
		platform.NewHosts("dipolehq.com"),
	)
	provider := &domain.CredentialProvider{
		ID:              uuid.New(),
		Type:            "oidc_id_token",
		ConfigEncrypted: encrypt(t, `{"subject":"workload","audience":"https://api.example.com"}`),
	}

	if _, err := broker.Retrieve(context.Background(), uuid.New(), provider, &domain.ClientEvidence{}, nil); err == nil {
		t.Fatal("expected an error when the tenant issuer could not be resolved")
	}
}

type multiTenantLookup struct {
	tenants map[uuid.UUID]*domain.Tenant
}

func (m *multiTenantLookup) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return m.tenants[id], nil
}
