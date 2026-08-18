// refresh_session_token_test.go tests the RefreshSessionToken RPC handler: requires an existing verified identity, never leaks internal errors, and returns the new token/expiry on success.
package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

func TestRefreshSessionToken_Unauthenticated_Denied(t *testing.T) {
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{},
		logger:    newTestLogger(),
	}

	resp, err := server.RefreshSessionToken(context.Background(), &pb.RefreshSessionTokenRequest{})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected denial when no verified proxy identity is present")
	}
}

func TestRefreshSessionToken_InternalErrorNotLeaked(t *testing.T) {
	sensitiveErr := fmt.Errorf("store session token: pq: connection to server at \"10.0.4.12\" failed")
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{refreshErr: sensitiveErr},
		logger:    newTestLogger(),
	}

	ctx := withVerifiedProxy(context.Background(), uuid.New(), uuid.New())
	resp, err := server.RefreshSessionToken(ctx, &pb.RefreshSessionTokenRequest{})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected refresh to fail")
	}
	if resp.ErrorMessage != "session token refresh failed" {
		t.Errorf("ErrorMessage = %q, want a fixed generic message", resp.ErrorMessage)
	}
}

func TestRefreshSessionToken_HappyPath_ReturnsNewTokenAndExpiry(t *testing.T) {
	expiresAt := time.Now().Add(1 * time.Hour)
	server := &PolicyEvaluationServer{
		registrar: &mockRegistrar{
			sessionToken:          "new-session-token",
			sessionTokenExpiresAt: expiresAt,
		},
		logger: newTestLogger(),
	}

	ctx := withVerifiedProxy(context.Background(), uuid.New(), uuid.New())
	resp, err := server.RefreshSessionToken(ctx, &pb.RefreshSessionTokenRequest{})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected success, got error: %s", resp.ErrorMessage)
	}
	if resp.SessionToken != "new-session-token" {
		t.Errorf("SessionToken = %q, want %q", resp.SessionToken, "new-session-token")
	}
	if resp.SessionTokenExpiresAt != expiresAt.Unix() {
		t.Errorf("SessionTokenExpiresAt = %d, want %d", resp.SessionTokenExpiresAt, expiresAt.Unix())
	}
}
