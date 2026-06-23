package injection

import (
	"net/http"
	"net/url"
	"testing"
)

func TestInjectCredential_Bearer(t *testing.T) {
	req := &http.Request{Header: make(http.Header), URL: &url.URL{}}
	cred := &Credential{Type: "bearer", Value: "my-token-123"}
	config := &InjectionConfig{Method: "http_bearer"}

	if err := InjectCredential(req, cred, config); err != nil {
		t.Fatal(err)
	}

	got := req.Header.Get("Authorization")
	want := "Bearer my-token-123"
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestInjectCredential_Basic(t *testing.T) {
	req := &http.Request{Header: make(http.Header), URL: &url.URL{}}
	cred := &Credential{Type: "basic", Username: "admin", Password: "secret"}
	config := &InjectionConfig{Method: "http_basic"}

	if err := InjectCredential(req, cred, config); err != nil {
		t.Fatal(err)
	}

	got := req.Header.Get("Authorization")
	// base64("admin:secret") = "YWRtaW46c2VjcmV0"
	want := "Basic YWRtaW46c2VjcmV0"
	if got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

func TestInjectCredential_CustomHeader(t *testing.T) {
	req := &http.Request{Header: make(http.Header), URL: &url.URL{}}
	cred := &Credential{Type: "apikey", Value: "sk_live_abc"}
	config := &InjectionConfig{Method: "http_header", HeaderName: "X-API-Key", Prefix: ""}

	if err := InjectCredential(req, cred, config); err != nil {
		t.Fatal(err)
	}

	got := req.Header.Get("X-API-Key")
	if got != "sk_live_abc" {
		t.Errorf("X-API-Key = %q, want %q", got, "sk_live_abc")
	}
}

func TestInjectCredential_APIKeyHeader(t *testing.T) {
	req := &http.Request{Header: make(http.Header), URL: &url.URL{}}
	cred := &Credential{Type: "apikey", Value: "key-xyz"}
	config := &InjectionConfig{Method: "api_key_header", HeaderName: "X-Custom-Auth", Prefix: "Token "}

	if err := InjectCredential(req, cred, config); err != nil {
		t.Fatal(err)
	}

	got := req.Header.Get("X-Custom-Auth")
	if got != "Token key-xyz" {
		t.Errorf("X-Custom-Auth = %q, want %q", got, "Token key-xyz")
	}
}

func TestInjectCredential_QueryParam(t *testing.T) {
	req := &http.Request{
		Header: make(http.Header),
		URL:    &url.URL{Path: "/api/data", RawQuery: "existing=true"},
	}
	cred := &Credential{Type: "apikey", Value: "my-key"}
	config := &InjectionConfig{Method: "api_key_query", QueryParam: "api_key"}

	if err := InjectCredential(req, cred, config); err != nil {
		t.Fatal(err)
	}

	got := req.URL.Query().Get("api_key")
	if got != "my-key" {
		t.Errorf("api_key = %q, want %q", got, "my-key")
	}
	// Existing param preserved
	if req.URL.Query().Get("existing") != "true" {
		t.Error("existing query param was lost")
	}
}

func TestInjectCredential_UnsupportedMethod(t *testing.T) {
	req := &http.Request{Header: make(http.Header), URL: &url.URL{}}
	cred := &Credential{Value: "x"}
	config := &InjectionConfig{Method: "sigv4"}

	err := InjectCredential(req, cred, config)
	if err == nil {
		t.Error("expected error for unsupported method, got nil")
	}
}
