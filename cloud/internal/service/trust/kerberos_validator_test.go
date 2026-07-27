/*
kerberos_validator_test.go tests Kerberos HMAC-SHA256 attestation verification.
*/
package trust

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

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
func (m *mockControllerRepoForKerberos) GetByDeviceCode(_ context.Context, _ uuid.UUID, _ string) (*domain.AgentController, error) {
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
			ID:           controllerID,
			SharedSecret: hex.EncodeToString(secret),
			Status:       domain.AgentStatusHealthy,
		},
	}

	v := NewKerberosValidator(repo)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[{"field":"principal","value":"user@CORP.LOCAL"}]}`),
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
	attest.Signature = "deadbeef" // Wrong signature

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:           controllerID,
			SharedSecret: hex.EncodeToString(secret),
		},
	}

	v := NewKerberosValidator(repo)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[]}`),
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
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix() // 10 minutes ago (> 5 min limit)
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", oldTimestamp, secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:           controllerID,
			SharedSecret: hex.EncodeToString(secret),
		},
	}

	v := NewKerberosValidator(repo)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[]}`),
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
	v := NewKerberosValidator(repo)

	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[]}`),
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
			ID:           controllerID,
			SharedSecret: hex.EncodeToString(secret),
		},
	}

	v := NewKerberosValidator(repo)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[{"field":"principal","value":"admin@CORP.LOCAL"}]}`),
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
	// Build attestation manually to put into environment map
	attest := buildKerberosAttestation("user@CORP.LOCAL", "CORP.LOCAL", "corp.local", time.Now().Unix(), secret)
	attest.ControllerID = controllerID.String()

	repo := &mockControllerRepoForKerberos{
		controller: &domain.AgentController{
			ID:           controllerID,
			SharedSecret: hex.EncodeToString(secret),
		},
	}

	v := NewKerberosValidator(repo)
	provider := &domain.TrustProvider{
		TenantID: uuid.New(),
		Config:   json.RawMessage(`{"match_rules":[{"field":"realm","value":"CORP.LOCAL"}]}`),
	}
	// Use environment map instead of KerberosAttestation field (backward compat)
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
