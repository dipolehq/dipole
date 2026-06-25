/*
Package credentials implements the Credential Broker that retrieves credentials
from Credential Providers.

CredentialBrokerImpl decrypts stored credential configs (AES-256-GCM), dispatches
to type-specific handlers, caches results, and returns credential values.
Decrypted values are never logged or stored beyond the Retrieve() call scope.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type CredentialBroker interface {
	Retrieve(ctx context.Context, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error)
}

type CredentialBrokerImpl struct {
	encryptionKey []byte
	cache         *CredentialCache
	issuer        *identity.IdentityIssuer

	dipoleAccessToken *DipoleAccessTokenHandler
	oidcIDToken       *OIDCIDTokenHandler
	jwtHandler        *JWTHandler
	jwtSVID           *JWTSVIDHandler
	x509SVID          *X509SVIDHandler
	oauthClientCreds  *OAuthClientCredsHandler
	oauthAuthCode     *OAuthAuthCodeHandler
	mcpUserToken      *MCPUserTokenHandler
	awsSts            *AWSStsHandler
	azureWIF          *AzureWIFHandler
	gcpWIF            *GCPWIFHandler
	claudeWIF         *ClaudeWIFHandler
	awsSecretsManager *AWSSecretsManagerHandler
	azureKeyVault     *AzureKeyVaultHandler
	vault             *VaultHandler
}

func NewCredentialBroker(encryptionKey []byte, issuer *identity.IdentityIssuer) *CredentialBrokerImpl {
	return &CredentialBrokerImpl{
		encryptionKey:     encryptionKey,
		cache:             NewCredentialCache(),
		issuer:            issuer,
		dipoleAccessToken: NewDipoleAccessTokenHandler(issuer),
		oidcIDToken:       NewOIDCIDTokenHandler(issuer),
		jwtHandler:        NewJWTHandler(issuer),
		jwtSVID:           NewJWTSVIDHandler(issuer),
		oauthClientCreds:  NewOAuthClientCredsHandler(),
		oauthAuthCode:     NewOAuthAuthCodeHandler(),
		mcpUserToken:      NewMCPUserTokenHandler(),
		awsSts:            NewAWSStsHandler(issuer),
		azureWIF:          NewAzureWIFHandler(issuer),
		gcpWIF:            NewGCPWIFHandler(issuer),
		claudeWIF:         NewClaudeWIFHandler(issuer),
		awsSecretsManager: NewAWSSecretsManagerHandler(issuer),
		azureKeyVault:     NewAzureKeyVaultHandler(issuer),
		vault:             NewVaultHandler(),
	}
}

func (b *CredentialBrokerImpl) Retrieve(ctx context.Context, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	// Check cache (skip for static types)
	if provider.Type != "api_key" && provider.Type != "username_password" {
		if cached, ok := b.cache.Get(provider.ID); ok {
			return cached, nil
		}
	}

	// Decrypt config
	configJSON, err := postgres.Decrypt(provider.ConfigEncrypted, b.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential config: %w", err)
	}

	// Dispatch to handler
	cred, err := b.dispatch(ctx, provider.Type, configJSON, evidence, trustClaims)
	if err != nil {
		return nil, err
	}

	// Cache result
	if ttl := CacheTTL(provider.Type, cred.ExpiresAt); ttl > 0 {
		b.cache.Set(provider.ID, cred, ttl)
	}

	return cred, nil
}

func (b *CredentialBrokerImpl) dispatch(ctx context.Context, providerType string, configJSON []byte, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	switch providerType {
	case "api_key":
		return b.retrieveAPIKey(configJSON)
	case "username_password":
		return b.retrieveUsernamePassword(configJSON)
	case "dipole_access_token":
		return b.dipoleAccessToken.Retrieve(ctx, configJSON, evidence)
	case "oidc_id_token":
		return b.oidcIDToken.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "json_web_token":
		return b.jwtHandler.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "jwt_svid":
		return b.jwtSVID.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "x509_svid":
		if b.x509SVID != nil {
			return b.x509SVID.Retrieve(ctx, configJSON, evidence, trustClaims)
		}
		return nil, fmt.Errorf("x509_svid: standalone CA not configured")
	case "oauth_client_credentials":
		return b.oauthClientCreds.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "oauth_authorization_code":
		return b.oauthAuthCode.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "mcp_user_based_access_token":
		return b.mcpUserToken.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "aws_sts_federation":
		return b.awsSts.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "azure_entra_wif":
		return b.azureWIF.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "gcp_wif":
		return b.gcpWIF.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "claude_wif":
		return b.claudeWIF.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "aws_secrets_manager":
		return b.awsSecretsManager.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "azure_key_vault":
		return b.azureKeyVault.Retrieve(ctx, configJSON, evidence, trustClaims)
	case "vault_client_token":
		return b.vault.Retrieve(ctx, configJSON, evidence, trustClaims)
	default:
		return nil, fmt.Errorf("credential provider type %q not implemented", providerType)
	}
}

func (b *CredentialBrokerImpl) retrieveAPIKey(configJSON []byte) (*domain.Credential, error) {
	var config struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("parse api key config: %w", err)
	}
	return &domain.Credential{Type: "bearer", Value: config.APIKey}, nil
}

func (b *CredentialBrokerImpl) retrieveUsernamePassword(configJSON []byte) (*domain.Credential, error) {
	var config struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("parse username/password config: %w", err)
	}
	return &domain.Credential{Type: "basic", Username: config.Username, Password: config.Password}, nil
}
