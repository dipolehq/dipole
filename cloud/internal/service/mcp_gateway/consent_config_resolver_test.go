package mcp_gateway

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"io"
	"testing"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func testEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

type mockCPRepo struct {
	cp  *domain.CredentialProvider
	err error
}

func (m *mockCPRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	panic("not used")
}
func (m *mockCPRepo) GetByID(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*domain.CredentialProvider, error) {
	return m.cp, m.err
}
func (m *mockCPRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListOptions) ([]domain.CredentialProvider, int, error) {
	panic("not used")
}
func (m *mockCPRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.CredentialProvider) error {
	panic("not used")
}
func (m *mockCPRepo) Delete(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	panic("not used")
}

func TestCPConsentConfigResolver_GetConsentConfig_Success(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	cpID := uuid.New()

	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}

	oauthConfig := map[string]string{
		"authorization_endpoint": "https://auth.jira.com/authorize",
		"token_endpoint":         "https://auth.jira.com/token",
		"client_id":              "jira-client-123",
		"client_secret":          "jira-secret-456",
		"scopes":                 "read:jira write:jira",
	}
	plaintext, _ := json.Marshal(oauthConfig)
	encrypted, err := testEncrypt(plaintext, encKey)
	if err != nil {
		t.Fatalf("encrypt test config: %v", err)
	}

	policyRepo := &mockAccessPolicyRepo{
		byProtocol: []domain.AccessPolicyWithRelations{
			{
				Policy: domain.AccessPolicy{
					ID:       uuid.New(),
					IsActive: true,
					CredentialProviderIDs: []domain.PolicyCredentialProvider{
						{CredentialProviderID: cpID},
					},
				},
				ServerWorkload: domain.ServerWorkload{
					ID:          serverWorkloadID,
					Name:        "jira",
					Host:        "jira.example.com",
					Port:        443,
					AppProtocol: domain.AppProtocolMCP,
				},
				ClientWorkload: domain.ClientWorkload{
					Identifiers: []domain.Identifier{
						{Type: domain.IdentifierTypeOIDCIDTokenAudience, Value: "https://gw.example.com/"},
					},
				},
			},
		},
	}

	cpRepo := &mockCPRepo{
		cp: &domain.CredentialProvider{
			ID:              cpID,
			Type:            "mcp_user_based_access_token",
			ConfigEncrypted: encrypted,
		},
	}

	resolver := NewCPConsentConfigResolver(policyRepo, cpRepo, encKey)

	cfg, err := resolver.GetConsentConfig(context.Background(), tenantID, serverWorkloadID, "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AuthorizationEndpoint != "https://auth.jira.com/authorize" {
		t.Errorf("expected authorization_endpoint 'https://auth.jira.com/authorize', got '%s'", cfg.AuthorizationEndpoint)
	}
	if cfg.TokenEndpoint != "https://auth.jira.com/token" {
		t.Errorf("expected token_endpoint 'https://auth.jira.com/token', got '%s'", cfg.TokenEndpoint)
	}
	if cfg.ClientID != "jira-client-123" {
		t.Errorf("expected client_id 'jira-client-123', got '%s'", cfg.ClientID)
	}
	if cfg.ClientSecret != "jira-secret-456" {
		t.Errorf("expected client_secret 'jira-secret-456', got '%s'", cfg.ClientSecret)
	}
	if cfg.Scopes != "read:jira write:jira" {
		t.Errorf("expected scopes 'read:jira write:jira', got '%s'", cfg.Scopes)
	}
	if cfg.GatewayCallbackURL != "https://gw.example.com/mcp/consent/callback" {
		t.Errorf("expected callback URL 'https://gw.example.com/mcp/consent/callback', got '%s'", cfg.GatewayCallbackURL)
	}
}

func TestCPConsentConfigResolver_GetConsentConfig_NoMatchingPolicy(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	otherServerID := uuid.New()

	policyRepo := &mockAccessPolicyRepo{
		byProtocol: []domain.AccessPolicyWithRelations{
			{
				Policy: domain.AccessPolicy{
					ID: uuid.New(),
					CredentialProviderIDs: []domain.PolicyCredentialProvider{
						{CredentialProviderID: uuid.New()},
					},
				},
				ServerWorkload: domain.ServerWorkload{
					ID:          otherServerID,
					AppProtocol: domain.AppProtocolMCP,
				},
			},
		},
	}

	cpRepo := &mockCPRepo{}
	encKey := make([]byte, 32)
	resolver := NewCPConsentConfigResolver(policyRepo, cpRepo, encKey)

	_, err := resolver.GetConsentConfig(context.Background(), tenantID, serverWorkloadID, "https://gw.example.com/")
	if err == nil {
		t.Fatal("expected error for non-matching server workload")
	}
}

func TestCPConsentConfigResolver_GetConsentConfig_CallbackURLWithoutTrailingSlash(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	cpID := uuid.New()

	encKey := make([]byte, 32)
	for i := range encKey {
		encKey[i] = byte(i)
	}
	oauthConfig := map[string]string{
		"authorization_endpoint": "https://auth.example.com/authorize",
		"token_endpoint":         "https://auth.example.com/token",
		"client_id":              "client",
	}
	plaintext, _ := json.Marshal(oauthConfig)
	encrypted, _ := testEncrypt(plaintext, encKey)

	policyRepo := &mockAccessPolicyRepo{
		byProtocol: []domain.AccessPolicyWithRelations{
			{
				Policy: domain.AccessPolicy{
					CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: cpID}},
				},
				ServerWorkload: domain.ServerWorkload{ID: serverWorkloadID, AppProtocol: domain.AppProtocolMCP},
			},
		},
	}
	cpRepo := &mockCPRepo{cp: &domain.CredentialProvider{ID: cpID, ConfigEncrypted: encrypted}}

	resolver := NewCPConsentConfigResolver(policyRepo, cpRepo, encKey)

	cfg, err := resolver.GetConsentConfig(context.Background(), tenantID, serverWorkloadID, "https://gw.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.GatewayCallbackURL != "https://gw.example.com/mcp/consent/callback" {
		t.Errorf("expected 'https://gw.example.com/mcp/consent/callback', got '%s'", cfg.GatewayCallbackURL)
	}
}

func TestCPConsentConfigResolver_GetConsentConfig_UnencryptedConfig(t *testing.T) {
	tenantID := uuid.New()
	serverWorkloadID := uuid.New()
	cpID := uuid.New()

	oauthConfig := map[string]string{
		"authorization_endpoint": "https://auth.example.com/authorize",
		"token_endpoint":         "https://auth.example.com/token",
		"client_id":              "test-client",
	}
	plaintext, _ := json.Marshal(oauthConfig)

	policyRepo := &mockAccessPolicyRepo{
		byProtocol: []domain.AccessPolicyWithRelations{
			{
				Policy: domain.AccessPolicy{
					CredentialProviderIDs: []domain.PolicyCredentialProvider{{CredentialProviderID: cpID}},
				},
				ServerWorkload: domain.ServerWorkload{ID: serverWorkloadID, AppProtocol: domain.AppProtocolMCP},
			},
		},
	}
	cpRepo := &mockCPRepo{cp: &domain.CredentialProvider{ID: cpID, ConfigEncrypted: plaintext}}

	resolver := NewCPConsentConfigResolver(policyRepo, cpRepo, nil)

	cfg, err := resolver.GetConsentConfig(context.Background(), tenantID, serverWorkloadID, "https://gw.example.com/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ClientID != "test-client" {
		t.Errorf("expected client_id 'test-client', got '%s'", cfg.ClientID)
	}
}

func (m *mockCPRepo) CountInactive(_ context.Context, _ uuid.UUID) int { return 0 }
