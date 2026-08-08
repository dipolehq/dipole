package mcp

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type mockMCPClientRepo struct {
	clients map[uuid.UUID]*RegisteredClient
}

func newMockClientRepo() *mockMCPClientRepo {
	return &mockMCPClientRepo{clients: make(map[uuid.UUID]*RegisteredClient)}
}

func (r *mockMCPClientRepo) Create(_ context.Context, client *RegisteredClient) error {
	r.clients[client.ID] = client
	return nil
}

func (r *mockMCPClientRepo) GetByID(_ context.Context, tenantID uuid.UUID, id uuid.UUID) (*RegisteredClient, error) {
	c, ok := r.clients[id]
	if !ok {
		return nil, ErrClientNotFound
	}
	if c.TenantID != tenantID {
		return nil, ErrClientNotFound
	}
	return c, nil
}

func TestDCR_ValidRequest_ReturnsClientID(t *testing.T) {
	svc := NewDCRService(newMockClientRepo())
	tenantID := uuid.New()

	resp, err := svc.Register(context.Background(), tenantID, &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ClientID == "" {
		t.Fatal("expected non-empty client_id")
	}
	if _, err := uuid.Parse(resp.ClientID); err != nil {
		t.Fatalf("client_id is not valid UUID: %v", err)
	}
	if resp.TokenEndpointAuthMethod != "none" {
		t.Errorf("expected token_endpoint_auth_method=none, got %s", resp.TokenEndpointAuthMethod)
	}
}

func TestDCR_MissingRedirectURIs_Error(t *testing.T) {
	svc := NewDCRService(newMockClientRepo())

	_, err := svc.Register(context.Background(), uuid.New(), &DCRRequest{
		RedirectURIs:  []string{},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"code"},
	})
	if err != ErrMissingRedirectURIs {
		t.Fatalf("expected ErrMissingRedirectURIs, got %v", err)
	}
}

func TestDCR_MissingGrantType_Error(t *testing.T) {
	svc := NewDCRService(newMockClientRepo())

	_, err := svc.Register(context.Background(), uuid.New(), &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"implicit"},
		ResponseTypes: []string{"code"},
	})
	if err != ErrMissingGrantType {
		t.Fatalf("expected ErrMissingGrantType, got %v", err)
	}
}

func TestDCR_MissingResponseType_Error(t *testing.T) {
	svc := NewDCRService(newMockClientRepo())

	_, err := svc.Register(context.Background(), uuid.New(), &DCRRequest{
		RedirectURIs:  []string{"http://localhost:7777/oauth/callback"},
		GrantTypes:    []string{"authorization_code"},
		ResponseTypes: []string{"token"},
	})
	if err != ErrMissingResponseType {
		t.Fatalf("expected ErrMissingResponseType, got %v", err)
	}
}

// RFC 7591 §2: when grant_types and response_types are omitted, defaults are
// authorization_code and code respectively. Claude.ai sends neither.
func TestDCR_OmittedGrantAndResponseTypes_DefaultsApplied(t *testing.T) {
	repo := newMockClientRepo()
	svc := NewDCRService(repo)

	resp, err := svc.Register(context.Background(), uuid.New(), &DCRRequest{
		RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback"},
		ClientName:   "Claude",
	})
	if err != nil {
		t.Fatalf("expected success with omitted grant/response types, got: %v", err)
	}
	if resp.ClientID == "" {
		t.Error("expected a client ID")
	}
	if len(resp.GrantTypes) != 1 || resp.GrantTypes[0] != "authorization_code" {
		t.Errorf("expected default grant_types=[authorization_code], got %v", resp.GrantTypes)
	}
}
