/*
edge_api_test.go tests the Edge API HTTP handlers using httptest with
mocked trust, policy, and tenant dependencies.
*/
package rest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

// --- Mocks ---

type mockEdgeTrust struct {
	result *domain.TrustProviderResult
	err    error
}

func (m *mockEdgeTrust) ValidateByID(_ uuid.UUID, _ uuid.UUID, _ *domain.ClientEvidence) (*domain.TrustProviderResult, error) {
	return m.result, m.err
}

type mockEdgePolicy struct {
	result *domain.EvaluationResult
	err    error
}

func (m *mockEdgePolicy) Evaluate(_ context.Context, _ uuid.UUID, _ *domain.ClientEvidence, _ *domain.ServerTarget) (*domain.EvaluationResult, error) {
	return m.result, m.err
}

type mockEdgeTenant struct {
	tenant *domain.Tenant
	err    error
}

func (m *mockEdgeTenant) GetBySubdomain(_ context.Context, _ string) (*domain.Tenant, error) {
	return m.tenant, m.err
}

// --- Helpers ---

func newTestEdgeAPI(t *testing.T, trustResult *domain.TrustProviderResult, policyResult *domain.EvaluationResult, tenant *domain.Tenant) (*EdgeAPIHandler, *identity.IdentityIssuer) {
	t.Helper()
	issuer, err := identity.NewIdentityIssuer("https://test-id.dipolehq.com")
	if err != nil {
		t.Fatal(err)
	}
	handler := NewEdgeAPIHandler(
		issuer,
		&mockEdgeTrust{result: trustResult},
		&mockEdgePolicy{result: policyResult},
		&mockEdgeTenant{tenant: tenant},
	)
	return handler, issuer
}

func validClientID() string {
	return "dipole:ap-south-1:acme:identity:github_idtoken:" + uuid.New().String()
}

// --- /edge/v1/auth tests ---

func TestHandleAuth_Success(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	trustResult := &domain.TrustProviderResult{Attested: true}
	handler, _ := newTestEdgeAPI(t, trustResult, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"clientId": validClientID(),
		"client":   map[string]any{"github": map[string]any{"identityToken": "eyJ.test.jwt"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["accessToken"] == nil || resp["accessToken"] == "" {
		t.Error("expected accessToken in response")
	}
	if resp["tokenType"] != "Bearer" {
		t.Errorf("expected tokenType 'Bearer', got '%v'", resp["tokenType"])
	}
	if resp["expiresIn"] != float64(3600) {
		t.Errorf("expected expiresIn 3600, got %v", resp["expiresIn"])
	}
}

func TestHandleAuth_RejectsAuthorizationHeader(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, _ := newTestEdgeAPI(t, &domain.TrustProviderResult{Attested: true}, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"clientId": validClientID(),
		"client":   map[string]any{"github": map[string]any{"identityToken": "jwt"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuth_InvalidClientID(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, _ := newTestEdgeAPI(t, &domain.TrustProviderResult{Attested: true}, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"clientId": "invalid-format",
		"client":   map[string]any{"github": map[string]any{"identityToken": "jwt"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAuth_TrustValidationFails(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	trustResult := &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}
	handler, _ := newTestEdgeAPI(t, trustResult, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"clientId": validClientID(),
		"client":   map[string]any{"github": map[string]any{"identityToken": "bad-jwt"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuth_TenantNotFound(t *testing.T) {
	handler, _ := newTestEdgeAPI(t, &domain.TrustProviderResult{Attested: true}, nil, nil)

	body, _ := json.Marshal(map[string]any{
		"clientId": validClientID(),
		"client":   map[string]any{"github": map[string]any{"identityToken": "jwt"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleAuth_InvalidJSON(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, _ := newTestEdgeAPI(t, &domain.TrustProviderResult{Attested: true}, nil, tenant)

	req := httptest.NewRequest(http.MethodPost, "/edge/v1/auth", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	handler.HandleAuth(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// --- /edge/v1/credentials tests ---

func TestHandleCredentials_Success(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	policyResult := &domain.EvaluationResult{
		Allowed:    true,
		Credential: &domain.Credential{Type: "api_key", Value: "sk_live_123", ExpiresAt: time.Now().Add(1 * time.Hour).Unix()},
	}
	handler, issuer := newTestEdgeAPI(t, nil, policyResult, tenant)

	clientID := validClientID()
	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:  clientID,
		Audience: "edge-api",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"client":         map[string]any{"github": map[string]any{"identityToken": "jwt"}},
		"server":         map[string]any{"host": "api.stripe.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["credentialType"] != "ApiKey" {
		t.Errorf("expected credentialType 'ApiKey', got '%v'", resp["credentialType"])
	}
	data, _ := resp["data"].(map[string]any)
	if data["apiKey"] != "sk_live_123" {
		t.Errorf("expected apiKey 'sk_live_123', got '%v'", data["apiKey"])
	}
}

func TestHandleCredentials_MissingBearer(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, _ := newTestEdgeAPI(t, nil, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"server":         map[string]any{"host": "api.stripe.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleCredentials_InvalidToken(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, _ := newTestEdgeAPI(t, nil, nil, tenant)

	body, _ := json.Marshal(map[string]any{
		"server":         map[string]any{"host": "api.stripe.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestHandleCredentials_PolicyDenied(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	policyResult := &domain.EvaluationResult{Allowed: false, DenyReason: "NoMatchingPolicy"}
	handler, issuer := newTestEdgeAPI(t, nil, policyResult, tenant)

	clientID := validClientID()
	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:  clientID,
		Audience: "edge-api",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"client":         map[string]any{"github": map[string]any{"identityToken": "jwt"}},
		"server":         map[string]any{"host": "forbidden.api.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["credentialType"] != "Unknown" {
		t.Errorf("expected credentialType 'Unknown' for denied policy, got '%v'", resp["credentialType"])
	}
}

func TestHandleCredentials_MalformedSubClaim_RejectedNotPanics(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, issuer := newTestEdgeAPI(t, nil, nil, tenant)

	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:  "not-a-valid-dipole-client-id",
		Audience: "edge-api",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"server":         map[string]any{"host": "api.stripe.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for a malformed sub claim (not a panic), got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCredentials_ExpiredToken(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	handler, issuer := newTestEdgeAPI(t, nil, nil, tenant)

	clientID := validClientID()
	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:  clientID,
		Audience: "edge-api",
		Lifetime: -1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"server":         map[string]any{"host": "api.stripe.com", "port": 443},
		"credentialType": "ApiKey",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- buildEdgeEvidence tests ---

func TestBuildEdgeEvidence_Certificate_WiresAttestation(t *testing.T) {
	sigBytes := []byte{0x01, 0x02, 0x03, 0xFF}
	client := edgeClientBlock{
		Certificate: &edgeCertificateAttn{
			CertificatePEM: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			Timestamp:      1700000000,
			Signature:      base64.StdEncoding.EncodeToString(sigBytes),
		},
	}

	evidence := buildEdgeEvidence(client, "certificate")

	if evidence.CertificateAttestation == nil {
		t.Fatal("expected CertificateAttestation to be populated")
	}
	if evidence.CertificateAttestation.CertificatePEM != client.Certificate.CertificatePEM {
		t.Errorf("CertificatePEM mismatch: got %q", evidence.CertificateAttestation.CertificatePEM)
	}
	if evidence.CertificateAttestation.Timestamp != 1700000000 {
		t.Errorf("Timestamp mismatch: got %d", evidence.CertificateAttestation.Timestamp)
	}
	if !bytes.Equal(evidence.CertificateAttestation.Signature, sigBytes) {
		t.Errorf("Signature mismatch: got %x, want %x", evidence.CertificateAttestation.Signature, sigBytes)
	}
}

func TestBuildEdgeEvidence_Certificate_MalformedSignatureBase64_LeavesAttestationNil(t *testing.T) {
	client := edgeClientBlock{
		Certificate: &edgeCertificateAttn{
			CertificatePEM: "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----",
			Timestamp:      1700000000,
			Signature:      "not-valid-base64!!!",
		},
	}

	evidence := buildEdgeEvidence(client, "certificate")

	if evidence.CertificateAttestation != nil {
		t.Error("expected CertificateAttestation to remain nil for malformed signature base64, letting CertificateValidator fail closed with NoDataFound")
	}
}

func TestBuildEdgeEvidence_Certificate_MissingBlock_NoAttestation(t *testing.T) {
	client := edgeClientBlock{}

	evidence := buildEdgeEvidence(client, "certificate")

	if evidence.CertificateAttestation != nil {
		t.Error("expected CertificateAttestation to be nil when no certificate block is present")
	}
}

func TestHandleCredentials_AwsStsFederation(t *testing.T) {
	tenant := &domain.Tenant{ID: uuid.New(), Subdomain: "acme"}
	policyResult := &domain.EvaluationResult{
		Allowed: true,
		Credential: &domain.Credential{
			Type:               "aws_sts",
			AWSAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE",
			AWSSessionToken:    "FwoGZXIvYXdzEBY...",
			ExpiresAt:          time.Now().Add(1 * time.Hour).Unix(),
		},
	}
	handler, issuer := newTestEdgeAPI(t, nil, policyResult, tenant)

	clientID := validClientID()
	token, err := issuer.IssueToken(identity.IssuanceRequest{
		Subject:  clientID,
		Audience: "edge-api",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{
		"client":         map[string]any{"github": map[string]any{"identityToken": "jwt"}},
		"server":         map[string]any{"host": "sts.amazonaws.com", "port": 443},
		"credentialType": "AwsStsFederation",
	})
	req := httptest.NewRequest(http.MethodPost, "/edge/v1/credentials", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.HandleCredentials(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["credentialType"] != "AwsStsFederation" {
		t.Errorf("expected 'AwsStsFederation', got '%v'", resp["credentialType"])
	}
	data, _ := resp["data"].(map[string]any)
	if data["awsAccessKeyId"] != "AKIAIOSFODNN7EXAMPLE" {
		t.Error("expected correct awsAccessKeyId")
	}
	if data["awsSecretAccessKey"] != "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLE" {
		t.Error("expected correct awsSecretAccessKey")
	}
	if data["awsSessionToken"] != "FwoGZXIvYXdzEBY..." {
		t.Error("expected correct awsSessionToken")
	}
}
