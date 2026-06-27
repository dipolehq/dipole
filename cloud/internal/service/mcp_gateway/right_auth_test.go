package mcp_gateway

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRightAuth_PerUserIsolation(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "alice-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}
	store.tokens["bob@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "bob-jira-token",
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, "https://gw.dipolehq.com/")

	// Alice gets Alice's token
	cred, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", tenantID, "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "alice-jira-token" {
		t.Errorf("expected alice-jira-token, got %s", cred.AccessToken)
	}

	// Bob gets Bob's token
	cred, err = rightAuth.GetCredential(context.Background(), "bob@corp.com", tenantID, "jira.example.com", 443)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cred.AccessToken != "bob-jira-token" {
		t.Errorf("expected bob-jira-token, got %s", cred.AccessToken)
	}
}

func TestRightAuth_FirstTimeAccess_ConsentRequired(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")

	_, err := rightAuth.GetCredential(context.Background(), "newuser@corp.com", tenantID, "jira.example.com", 443)
	if err != ErrConsentRequired {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}
}

func TestRightAuth_NoPolicyForServer_Denied(t *testing.T) {
	tenantID := uuid.New()
	policyFinder := &mockRightSidePolicyFinder{policies: nil}
	rightAuth := NewRightSideAuth(policyFinder, newMockTokenStore(), "https://gw.dipolehq.com/")

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", tenantID, "unknown.example.com", 443)
	if err != ErrNoPolicyForServer {
		t.Fatalf("expected ErrNoPolicyForServer, got %v", err)
	}
}

func TestRightAuth_ExpiredToken_Error(t *testing.T) {
	serverID := uuid.New()
	tenantID := uuid.New()

	store := newMockTokenStore()
	store.tokens["alice@corp.com:"+serverID.String()] = &UserTokenEntry{
		AccessToken: "expired-token",
		ExpiresAt:   time.Now().UTC().Add(-1 * time.Hour),
	}

	policyFinder := &mockRightSidePolicyFinder{
		policies: []RightSidePolicy{
			{
				ServerWorkload: ServerInfo{ID: serverID, Name: "jira", Host: "jira.example.com", Port: 443},
				CredentialType: "mcp_user_based_access_token",
			},
		},
	}

	rightAuth := NewRightSideAuth(policyFinder, store, "https://gw.dipolehq.com/")

	_, err := rightAuth.GetCredential(context.Background(), "alice@corp.com", tenantID, "jira.example.com", 443)
	if err != ErrCredentialExpired {
		t.Fatalf("expected ErrCredentialExpired, got %v", err)
	}
}
