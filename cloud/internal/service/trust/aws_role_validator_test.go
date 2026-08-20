package trust

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

const stsValidResponse = `<GetCallerIdentityResponse>
  <GetCallerIdentityResult>
    <Arn>arn:aws:sts::123456789012:assumed-role/my-role/i-0abc</Arn>
    <Account>123456789012</Account>
    <UserId>AROAEXAMPLE:i-0abc</UserId>
  </GetCallerIdentityResult>
</GetCallerIdentityResponse>`

func TestAWSRoleValidator_ValidSTS_Attested(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, stsValidResponse)
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s", // region gets substituted but URL points to mock
	}

	cfg, _ := json.Marshal(struct{}{})

	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "accountId", Value: "123456789012"}},
	}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{
					"Authorization": "AWS4-HMAC-SHA256 Credential=AKIA.../20260625/ap-south-1/sts/aws4_request, SignedHeaders=content-type;host;x-amz-date;x-amz-security-token, Signature=abc123",
					"Content-Type":  "application/x-www-form-urlencoded; charset=utf-8",
				},
				Region: "ap-south-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested, got reason: %s", result.Reason)
	}
	if result.Claims == nil {
		t.Fatal("expected Claims to be populated on successful attestation")
	}
	if result.Claims["accountId"] != "123456789012" {
		t.Errorf("Claims[accountId] = %v, want 123456789012", result.Claims["accountId"])
	}
	if result.Claims["arn"] != "arn:aws:sts::123456789012:assumed-role/my-role/i-0abc" {
		t.Errorf("Claims[arn] = %v, want the assumed-role ARN", result.Claims["arn"])
	}
}

func TestAWSRoleValidator_STS403_InvalidSignature(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s",
	}

	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{"Authorization": "invalid"},
				Region:        "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestAWSRoleValidator_MatchRuleFailed(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, stsValidResponse)
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s",
	}

	cfg, _ := json.Marshal(struct{}{})

	provider := &domain.TrustProvider{
		ID:         uuid.New(),
		Config:     cfg,
		MatchRules: []domain.MatchRule{{Attribute: "accountId", Value: "999999999999"}},
	}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{"Authorization": "AWS4-HMAC-SHA256 ..."},
				Region:        "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "MatchRuleFailed" {
		t.Fatalf("expected MatchRuleFailed, got %s", result.Reason)
	}
	if result.ActualValue != "123456789012" {
		t.Fatalf("expected actual 123456789012, got %s", result.ActualValue)
	}
}

func TestAWSRoleValidator_NoEvidence_NoDataFound(t *testing.T) {
	v := NewAWSRoleValidator()
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}

	result, err := v.Validate(context.Background(), provider, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "NoDataFound" {
		t.Fatalf("expected NoDataFound, got %s", result.Reason)
	}
}

func TestAWSRoleValidator_EmptyHeaders_NoDataFound(t *testing.T) {
	v := NewAWSRoleValidator()
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}

	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{},
				Region:        "",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != "NoDataFound" {
		t.Fatalf("expected NoDataFound, got %s", result.Reason)
	}
}

func TestAWSRoleValidator_STSUnreachable_InvalidSignature(t *testing.T) {
	v := &AWSRoleValidator{
		httpClient: &http.Client{Timeout: 1 * time.Second},
		stsURLFmt:  "https://localhost:1/%s", // unreachable
	}
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}

	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{"Authorization": "x"},
				Region:        "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestNewAWSRoleValidator_SSRFProtected(t *testing.T) {
	v := NewAWSRoleValidator()
	transport, ok := v.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected httpClient.Transport to be *http.Transport, got %T", v.httpClient.Transport)
	}
	if transport.DialContext == nil {
		t.Fatal("expected DialContext to be set to the SSRF-safe dialer")
	}

	// End-to-end proof: dialing a loopback address (what an
	// attacker-influenced region/host would resolve to in an SSRF
	// attempt) through this client's transport must be refused.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := v.httpClient.Get(server.URL)
	if err == nil {
		t.Fatal("expected request to loopback address to be refused by SSRF-safe transport")
	}
}

func TestAWSRoleValidator_InvalidRegion_Rejected(t *testing.T) {
	v := &AWSRoleValidator{
		httpClient: http.DefaultClient,
		stsURLFmt:  "https://sts.%s.amazonaws.com/",
	}
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}

	tests := []string{
		"us-east-1.attacker.com",
		"us-east-1#fragment",
		"us-east-1/../../admin",
		"US-EAST-1",
		"",
		"not-a-region-at-all-really",
	}

	for _, region := range tests {
		evidence := &domain.ClientEvidence{
			AWS: &domain.AWSEvidence{
				STSGetCallerIdentity: &domain.STSGetCallerIdentity{
					SignedHeaders: map[string]string{"Authorization": "x"},
					Region:        region,
				},
			},
		}

		result, err := v.Validate(context.Background(), provider, evidence, nil)
		if err != nil {
			t.Fatalf("region %q: unexpected error: %v", region, err)
		}
		if result.Attested {
			t.Errorf("region %q: expected denied, got attested", region)
		}
		if region == "" {
			// Empty region is caught by the earlier NoDataFound check,
			// not the region-format validation.
			if result.Reason != "NoDataFound" {
				t.Errorf("region %q: reason = %q, want NoDataFound", region, result.Reason)
			}
			continue
		}
		if result.Reason != "InvalidRegion" {
			t.Errorf("region %q: reason = %q, want InvalidRegion", region, result.Reason)
		}
	}
}

func TestAWSRoleValidator_ValidRegionFormats_PassValidation(t *testing.T) {
	// These are real AWS region shapes (standard, GovCloud, China
	// partitions) that must not be rejected by the region-format check.
	// The request will still fail downstream since there's no live STS
	// mock here, but the failure must not be InvalidRegion.
	regions := []string{"us-east-1", "ap-south-1", "eu-west-2", "us-gov-west-1", "cn-north-1"}

	for _, region := range regions {
		if !awsRegionPattern.MatchString(region) {
			t.Errorf("region %q: expected to match awsRegionPattern", region)
		}
	}
}

func TestAWSRoleValidator_ReplaySTS_OnlyForwardsAllowedHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, stsValidResponse)
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s",
	}

	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{
					"Authorization":        "AWS4-HMAC-SHA256 Credential=AKIA...",
					"X-Amz-Date":           "20260101T000000Z",
					"X-Amz-Security-Token": "token123",
					"Host":                 "attacker-controlled-host.example.com",
					"Content-Length":       "9999",
					"X-Custom-Header":      "should-not-be-forwarded",
				},
				Region: "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Attested {
		t.Fatalf("expected attested, got reason: %s", result.Reason)
	}

	if receivedHeaders.Get("Authorization") == "" {
		t.Error("expected Authorization header to be forwarded")
	}
	if receivedHeaders.Get("X-Amz-Date") == "" {
		t.Error("expected X-Amz-Date header to be forwarded")
	}
	if receivedHeaders.Get("X-Amz-Security-Token") == "" {
		t.Error("expected X-Amz-Security-Token header to be forwarded")
	}
	if receivedHeaders.Get("X-Custom-Header") != "" {
		t.Error("expected X-Custom-Header to NOT be forwarded (not in allowlist)")
	}
	// Host and Content-Length are controlled by the net/http client
	// itself (not settable via req.Header for an outbound request in the
	// way an attacker would need), but confirming they weren't smuggled
	// through via req.Header.Set is still worth asserting on the
	// allowlist's own behavior directly:
	if allowedSigV4Headers["Host"] {
		t.Error("Host must not be in the SigV4 header allowlist")
	}
	if allowedSigV4Headers["Content-Length"] {
		t.Error("Content-Length must not be in the SigV4 header allowlist")
	}
}

func TestAWSRoleValidator_STSMalformedXML_InvalidSignature(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<this is not valid xml")
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s",
	}
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{"Authorization": "x"},
				Region:        "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied for malformed XML response")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}

func TestAWSRoleValidator_STS500_InvalidSignature(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "<Error><Code>InternalFailure</Code></Error>")
	}))
	defer server.Close()

	v := &AWSRoleValidator{
		httpClient: server.Client(),
		stsURLFmt:  server.URL + "/%s",
	}
	cfg, _ := json.Marshal(struct{}{})
	provider := &domain.TrustProvider{ID: uuid.New(), Config: cfg}
	evidence := &domain.ClientEvidence{
		AWS: &domain.AWSEvidence{
			STSGetCallerIdentity: &domain.STSGetCallerIdentity{
				SignedHeaders: map[string]string{"Authorization": "x"},
				Region:        "us-east-1",
			},
		},
	}

	result, err := v.Validate(context.Background(), provider, evidence, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.Attested {
		t.Fatal("expected denied for STS 500 response")
	}
	if result.Reason != "InvalidSignature" {
		t.Fatalf("expected InvalidSignature, got %s", result.Reason)
	}
}
