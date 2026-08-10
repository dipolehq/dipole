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
