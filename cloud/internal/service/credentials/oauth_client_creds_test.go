// oauth_client_creds_test.go tests the OAuth Client Credentials handler's token fetch, credential styles, and error handling.
package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestOAuthClientCreds_FetchesToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Fatalf("wrong content-type: %s", r.Header.Get("Content-Type"))
		}
		r.ParseForm()
		if r.FormValue("grant_type") != "client_credentials" {
			t.Fatalf("grant_type = %s", r.FormValue("grant_type"))
		}
		if r.FormValue("client_id") != "my-id" {
			t.Fatalf("client_id = %s", r.FormValue("client_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"access_token":"tok-123","expires_in":3600}`)
	}))
	defer server.Close()

	handler := &OAuthClientCredsHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(oauthClientCredsConfig{
		TokenURL:     server.URL,
		ClientID:     "my-id",
		ClientSecret: "my-secret",
		Scopes:       "read write",
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Value != "tok-123" {
		t.Errorf("value = %q, want tok-123", cred.Value)
	}
	if cred.ExpiresAt == 0 {
		t.Error("ExpiresAt should be set")
	}
}

func TestOAuthClientCreds_AuthorizationHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Fatal("missing Authorization header")
		}
		if r.FormValue("client_id") != "" {
			t.Fatal("client_id should NOT be in form body for auth_header style")
		}
		fmt.Fprint(w, `{"access_token":"header-tok","expires_in":1800}`)
	}))
	defer server.Close()

	handler := &OAuthClientCredsHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(oauthClientCredsConfig{
		TokenURL:        server.URL,
		ClientID:        "cid",
		ClientSecret:    "csec",
		CredentialStyle: "authorization_header",
	})

	cred, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cred.Value != "header-tok" {
		t.Errorf("value = %q, want header-tok", cred.Value)
	}
}

func TestOAuthClientCreds_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"invalid_client"}`)
	}))
	defer server.Close()

	handler := &OAuthClientCredsHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(oauthClientCredsConfig{TokenURL: server.URL, ClientID: "x", ClientSecret: "y"})

	_, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err == nil {
		t.Error("expected error for 400 response")
	}
}

func TestOAuthClientCreds_CachingBehavior(t *testing.T) {
	var callCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		fmt.Fprint(w, `{"access_token":"cached-tok","expires_in":3600}`)
	}))
	defer server.Close()

	handler := &OAuthClientCredsHandler{httpClient: server.Client()}
	cfg, _ := json.Marshal(oauthClientCredsConfig{TokenURL: server.URL, ClientID: "x", ClientSecret: "y"})

	cred1, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	cred2, err := handler.Retrieve(context.Background(), cfg, &domain.ClientEvidence{}, nil)
	if err != nil {
		t.Fatal(err)
	}

	if cred1.Value != cred2.Value {
		t.Error("both calls should return same token value")
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 HTTP calls (caching is broker's job), got %d", callCount.Load())
	}
}
