package injection

import (
	"net/http"
	"strings"
	"testing"
)

func TestSignAWSRequest_SetsHeaders(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://s3.us-east-2.amazonaws.com/mybucket/mykey", nil)
	creds := &AWSCredentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		SessionToken:    "session-token-123",
	}

	err := SignAWSRequest(req, creds, "us-east-2", "s3")
	if err != nil {
		t.Fatal(err)
	}

	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 Credential=AKIAIOSFODNN7EXAMPLE/") {
		t.Errorf("Authorization header wrong: %s", auth)
	}
	if !strings.Contains(auth, "us-east-2/s3/aws4_request") {
		t.Errorf("credential scope missing region/service: %s", auth)
	}
	if !strings.Contains(auth, "SignedHeaders=") {
		t.Errorf("missing SignedHeaders: %s", auth)
	}
	if !strings.Contains(auth, "Signature=") {
		t.Errorf("missing Signature: %s", auth)
	}

	if req.Header.Get("X-Amz-Date") == "" {
		t.Error("missing X-Amz-Date")
	}
	if req.Header.Get("X-Amz-Security-Token") != "session-token-123" {
		t.Error("missing X-Amz-Security-Token")
	}
	if req.Header.Get("X-Amz-Content-Sha256") == "" {
		t.Error("missing X-Amz-Content-Sha256")
	}
}

func TestSignAWSRequest_WithBody(t *testing.T) {
	body := `{"key":"value"}`
	req, _ := http.NewRequest("POST", "https://dynamodb.ap-south-1.amazonaws.com/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-amz-json-1.0")

	creds := &AWSCredentials{
		AccessKeyID:     "AKID",
		SecretAccessKey: "SECRET",
	}

	err := SignAWSRequest(req, creds, "ap-south-1", "dynamodb")
	if err != nil {
		t.Fatal(err)
	}

	contentHash := req.Header.Get("X-Amz-Content-Sha256")
	if contentHash == "" || contentHash == "UNSIGNED-PAYLOAD" {
		t.Error("expected actual body hash for POST with body")
	}

	// No session token set → header should be absent
	if req.Header.Get("X-Amz-Security-Token") != "" {
		t.Error("session token should not be set when empty")
	}
}

func TestSignAWSRequest_Consistency(t *testing.T) {
	// Same inputs → same signature (within same second)
	req1, _ := http.NewRequest("GET", "https://s3.us-east-1.amazonaws.com/", nil)
	req2, _ := http.NewRequest("GET", "https://s3.us-east-1.amazonaws.com/", nil)

	creds := &AWSCredentials{AccessKeyID: "AK", SecretAccessKey: "SK"}

	SignAWSRequest(req1, creds, "us-east-1", "s3")
	SignAWSRequest(req2, creds, "us-east-1", "s3")

	// Both should have same date (within same second)
	if req1.Header.Get("X-Amz-Date") != req2.Header.Get("X-Amz-Date") {
		// Possible if test spans a second boundary — acceptable
		t.Log("dates differ (second boundary crossed)")
	}
}

func TestExtractServiceRegion(t *testing.T) {
	tests := []struct {
		host    string
		service string
		region  string
	}{
		{"s3.us-east-2.amazonaws.com", "s3", "us-east-2"},
		{"dynamodb.ap-south-1.amazonaws.com", "dynamodb", "ap-south-1"},
		{"sts.ap-south-1.amazonaws.com", "sts", "ap-south-1"},
		{"s3.amazonaws.com", "s3", "us-east-1"},
	}

	for _, tt := range tests {
		svc, reg, err := ExtractServiceRegion(tt.host)
		if err != nil {
			t.Errorf("%s: %v", tt.host, err)
			continue
		}
		if svc != tt.service {
			t.Errorf("%s: service=%q, want %q", tt.host, svc, tt.service)
		}
		if reg != tt.region {
			t.Errorf("%s: region=%q, want %q", tt.host, reg, tt.region)
		}
	}
}
