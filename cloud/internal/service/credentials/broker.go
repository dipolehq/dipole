/*
Package credentials implements the Credential Broker that retrieves credentials
from Credential Providers.

CredentialBrokerImpl decrypts stored credential configs (AES-256-GCM), dispatches
to type-specific handlers, caches results, and returns credential values.
Decrypted values are never logged or stored beyond the Retrieve() call scope.

tenantID is required on Retrieve because several credential types mint a token
signed by us, and the iss claim must be the requesting tenant's identity host.
AWS STS, Azure Entra and GCP WIF all fetch OIDC discovery from iss and reject a
mismatch. One process serves every tenant, so the issuer URL cannot be fixed at
startup.

Cache entries are keyed by credential provider ID, and a provider belongs to
exactly one tenant, so a cached credential can never cross tenants. Static types
(api_key, username_password) bypass the cache since they never expire.

selfSignedCredentialTypes lists types where we mint the token and need the
tenant's identity issuer URL. Everything else either returns a stored secret or
fetches a token from an external provider. aws_secrets_manager and
azure_key_vault don't sign tokens directly but authenticate to the secret store
by delegating to the STS/Azure WIF handlers, which do.

tenantIssuerURL fails closed for self-signed types: a wrong iss would be
accepted here and then rejected opaquely by AWS/Azure/GCP, which is far harder
to diagnose than an explicit error. An empty IssuerURL would silently fall back
to the platform issuer, so we refuse to build one from an invalid subdomain.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/platform"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
	"github.com/google/uuid"
)

type CredentialBroker interface {
	Retrieve(ctx context.Context, tenantID uuid.UUID, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error)
}

type TenantLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
}

type CredentialBrokerImpl struct {
	encryptionKey []byte
	cache         *CredentialCache
	issuer        *identity.IdentityIssuer
	tenants       TenantLookup
	hosts         *platform.Hosts

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

func NewCredentialBroker(encryptionKey []byte, issuer *identity.IdentityIssuer, tenants TenantLookup, hosts *platform.Hosts) *CredentialBrokerImpl {
	return &CredentialBrokerImpl{
		encryptionKey:     encryptionKey,
		cache:             NewCredentialCache(),
		issuer:            issuer,
		tenants:           tenants,
		hosts:             hosts,
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

func (b *CredentialBrokerImpl) Retrieve(ctx context.Context, tenantID uuid.UUID, provider *domain.CredentialProvider, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	if provider.Type != "api_key" && provider.Type != "username_password" {
		if cached, ok := b.cache.Get(provider.ID); ok {
			return cached, nil
		}
	}

	configJSON, err := crypto.Decrypt(provider.ConfigEncrypted, b.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt credential config: %w", err)
	}

	issuerURL, err := b.tenantIssuerURL(ctx, tenantID, provider.Type)
	if err != nil {
		return nil, err
	}

	cred, err := b.dispatch(ctx, provider.Type, configJSON, issuerURL, evidence, trustClaims)
	if err != nil {
		return nil, err
	}

	if ttl := CacheTTL(provider.Type, cred.ExpiresAt); ttl > 0 {
		b.cache.Set(provider.ID, cred, ttl)
	}

	return cred, nil
}

var selfSignedCredentialTypes = map[string]bool{
	"dipole_access_token": true,
	"oidc_id_token":       true,
	"json_web_token":      true,
	"jwt_svid":            true,
	"aws_sts_federation":  true,
	"azure_entra_wif":     true,
	"gcp_wif":             true,
	"claude_wif":          true,
	"aws_secrets_manager": true,
	"azure_key_vault":     true,
}

func (b *CredentialBrokerImpl) tenantIssuerURL(ctx context.Context, tenantID uuid.UUID, providerType string) (string, error) {
	if !selfSignedCredentialTypes[providerType] {
		return "", nil
	}
	if b.tenants == nil || b.hosts == nil {
		return "", fmt.Errorf("credential type %q signs a token but the broker has no tenant issuer resolution configured", providerType)
	}

	tenant, err := b.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("resolve tenant for token issuer: %w", err)
	}
	if tenant == nil {
		return "", fmt.Errorf("resolve tenant for token issuer: tenant %s not found", tenantID)
	}

	issuerURL := b.hosts.IdentityIssuerURL(tenant.Subdomain)
	if issuerURL == "" {
		return "", fmt.Errorf(
			"cannot build token issuer URL for tenant %s: subdomain %q is not a valid hostname label",
			tenantID, tenant.Subdomain)
	}
	return issuerURL, nil
}

func (b *CredentialBrokerImpl) dispatch(ctx context.Context, providerType string, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, trustClaims map[string]interface{}) (*domain.Credential, error) {
	switch providerType {
	case "api_key":
		return b.retrieveAPIKey(configJSON)
	case "username_password":
		return b.retrieveUsernamePassword(configJSON)
	case "dipole_access_token":
		return b.dipoleAccessToken.Retrieve(ctx, configJSON, issuerURL, evidence)
	case "oidc_id_token":
		return b.oidcIDToken.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "json_web_token":
		return b.jwtHandler.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "jwt_svid":
		return b.jwtSVID.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
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
		return b.awsSts.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "azure_entra_wif":
		return b.azureWIF.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "gcp_wif":
		return b.gcpWIF.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "claude_wif":
		return b.claudeWIF.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "aws_secrets_manager":
		return b.awsSecretsManager.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "azure_key_vault":
		return b.azureKeyVault.Retrieve(ctx, configJSON, issuerURL, evidence, trustClaims)
	case "vault_client_token":
		return b.vault.Retrieve(ctx, configJSON, evidence, trustClaims)
	default:
		return nil, fmt.Errorf("credential provider type %q not implemented", providerType)
	}
}

func (b *CredentialBrokerImpl) retrieveAPIKey(configJSON []byte) (*domain.Credential, error) {
	var config struct {
		APIKey string `json:"apiKey"`
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
