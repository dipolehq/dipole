// kerberos_validator_test.go tests Kerberos HMAC-SHA256 attestation verification.
package trust

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
)

var testEncryptionKey = bytes.Repeat([]byte{0x24}, 32)

func encryptSecretForTest(t *testing.T, secret []byte) []byte {
	t.Helper()
	encrypted, err := crypto.Encrypt([]byte(hex.EncodeToString(secret)), testEncryptionKey)
	if err != nil {
		t.Fatalf("encrypt test secret: %v", err)
	}
	return encrypted
}

type mockControllerRepoForKerberos struct {
	controller *domain.AgentController
	err        error
}

func (m *mockControllerRepoForKerberos) Create(_ context.Context, _ uuid.UUID, _ *domain.AgentController) error {
	return nil
}
func (m *mockControllerRepoForKerberos) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.AgentController, error) {
	return m.controller, m.err
}
func (m *mockControllerRepoForKerberos) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.AgentController, int, error) {
	return nil, 0, nil
}
func (m *mockControllerRepoForKerberos) Update(_ context.Context, _ uuid.UUID, _ *domain.AgentController) error {
	return nil
}
func (m *mockControllerRepoForKerberos) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *mockControllerRepoForKerberos) ConsumeDeviceCode(_ context.Context, _ uuid.UUID, _ string, _ []byte) (*domain.AgentController, error) {
	return nil, nil
}

func buildKerberosAttestation(principal, realm, domainName string, timestamp int64, secret []byte) *domain.KerberosAttestation {
	payload := map[string]interface{}{
		"principal": principal,
		"realm":     realm,
		"domain":    domainName,
		"timestamp": timestamp,
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.StdEncoding.EncodeToString(payloadJSON)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	sig := hex.EncodeToString(mac.Sum(nil))

	return &domain.KerberosAttestation{
		Payload:      payloadB64,
		Signature:    sig,
		ControllerID: uuid.New().String(),
	}
}

func TestKerberosValidator_ValidAttestation_Attested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
			Status:                domain.AgentStatusHealthy,
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID:   uuid.New(),
		Config:     json.RawMessage(`{}`),
		MatchRules: []domain.MatchRule{{Attribute: "principal", Value: "user@CORP.LOCAL"}},
	}
	evidence := &domain.ClientEvidence{
		KerberosAttestation: attest,
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true, got reason: %s", result.Reason)
	}
}

func TestKerberosValidator_InvalidSignature_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()
	attest.Signature = "deadbeef"

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for wrong signature")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_ExpiredTimestamp_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", oldTimestamp, secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for expired timestamp")
	}
	if result.Reason != "AttestationExpired" {
		t.Errorf("expected reason 'AttestationExpired', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_NoAttestation_NotAttested(t *testing.T) {
	repo := &mockControllerRepoForKerberos{}
	v := NewKerberosValidator(repo, testEncryptionKey)

	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false with no attestation")
	}
	if result.Reason != "NoDataFound" {
		t.Errorf("expected 'NoDataFound', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_MatchRuleFailed_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID:   uuid.New(),
		Config:     json.RawMessage(`{}`),
		MatchRules: []domain.MatchRule{{Attribute: "principal", Value: "admin@CORP.LOCAL"}},
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for match rule failure")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Errorf("expected reason 'MatchRuleFailed', got '%s'", result.Reason)
	}
	if result.Attribute != "principal" {
		t.Errorf("expected attribute 'principal', got '%s'", result.Attribute)
	}
}

func TestKerberosValidator_LegacyEnvironmentFallback_Attested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID:   uuid.New(),
		Config:     json.RawMessage(`{}`),
		MatchRules: []domain.MatchRule{{Attribute: "realm", Value: "CORP.LOCAL"}},
	}
	evidence := &domain.ClientEvidence{
		Environment: map[string]string{
			"KERBEROS_ATTESTATION_PAYLOAD":   attest.Payload,
			"KERBEROS_ATTESTATION_SIGNATURE": attest.Signature,
			"KERBEROS_CONTROLLER_ID":         attest.ControllerID,
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Attested {
		t.Errorf("expected attested=true via legacy env fallback, got reason: %s", result.Reason)
	}
}

func TestKerberosValidator_NonexistentController_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = uuid.New().String()

	// Repo has no matching controller — GetByID returns (nil, nil), the
	// same not-found convention used elsewhere in this codebase's
	// repositories.
	repo := &mockControllerRepoForKerberos{controller: nil, err: nil}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for nonexistent controller")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_ControllerRepoError_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = uuid.New().String()

	// Repo lookup itself fails (e.g. a transient DB error) — must fail
	// closed exactly the same as a genuine not-found, not surface the
	// underlying error to the caller.
	repo := &mockControllerRepoForKerberos{controller: nil, err: fmt.Errorf("db unavailable")}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when controller repo returns an error")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_MissingSharedSecret_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()

	// Controller exists but has never completed registration/claim, so
	// it has no shared secret persisted (len(SharedSecretEncrypted) == 0).
	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: nil,
		},
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false when controller has no shared secret")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_MalformedPayloadBase64_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	// Craft an attestation whose signature is valid for the given
	// (invalid-base64) payload, so it reaches the base64-decode step
	// rather than being rejected earlier by the signature check.
	invalidPayload := "!!!not-valid-base64!!!"
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(invalidPayload))
	sig := hex.EncodeToString(mac.Sum(nil))

	attest := &domain.KerberosAttestation{
		Payload:      invalidPayload,
		Signature:    sig,
		ControllerID: controllerID.String(),
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for malformed base64 payload")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}

func TestKerberosValidator_MalformedPayloadJSON_NotAttested(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i)
	}

	controllerID := uuid.New()
	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:                    controllerID,
			SharedSecretEncrypted: encryptSecretForTest(t, secret),
		},
	}

	// Valid base64, but the decoded bytes are not valid JSON — signature
	// still checks out for whatever bytes were actually signed, so this
	// reaches the JSON-unmarshal step.
	payloadBytes := []byte("not valid json")
	payloadB64 := base64.StdEncoding.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	sig := hex.EncodeToString(mac.Sum(nil))

	attest := &domain.KerberosAttestation{
		Payload:      payloadB64,
		Signature:    sig,
		ControllerID: controllerID.String(),
	}

	v := NewKerberosValidator(repo, testEncryptionKey)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{}`),
	}
	evidence := &domain.ClientEvidence{KerberosAttestation: attest}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Attested {
		t.Error("expected attested=false for malformed JSON payload")
	}
	if result.Reason != "InvalidSignature" {
		t.Errorf("expected reason 'InvalidSignature', got '%s'", result.Reason)
	}
}
