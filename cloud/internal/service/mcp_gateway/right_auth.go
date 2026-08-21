// right_auth.go implements RightSideAuth, which evaluates Gateway-to-Server authorization and resolves the credential for a downstream MCP server.
package mcp_gateway

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNoPolicyForServer               = errors.New("no access policy found for this MCP server")
	ErrConsentRequired                 = errors.New("user authorization required for this MCP server")
	ErrCredentialExpired               = errors.New("credential expired and refresh failed")
	ErrInvalidTenantScope              = errors.New("request has no valid tenant scope")
	ErrCredentialProviderNotConfigured = errors.New("credential provider not configured for this Gateway")
)

type ConsentRequiredError struct {
	ServerWorkloadID uuid.UUID
	ConsentURL       string
}

func (e *ConsentRequiredError) Error() string { return ErrConsentRequired.Error() }
func (e *ConsentRequiredError) Unwrap() error { return ErrConsentRequired }

type ServerCredential struct {
	Authorization        string
	ServerWorkload       ServerInfo
	ResourceSetID        uuid.UUID
	PolicyID             uuid.UUID
	PolicyName           string
	CredentialType       string
	CredentialName       string
	CredentialProviderID uuid.UUID
}

type ServerInfo struct {
	ID   uuid.UUID
	Name string
	Host string
	Port int
	Path string
	TLS  bool
}

type UserTokenEntry struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type UserTokenStore interface {
	Get(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error)
	// Store saves a user's per-server OAuth token. credentialProviderID
	// records which Credential Provider issued this token (for
	// auditability); it is not part of the row's identity — a token is
	// looked up by tenant + user + server alone, matching Get above, so
	// re-consenting under a different Credential Provider for the same
	// tenant/user/server still updates the existing row rather than
	// creating a duplicate.
	Store(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, credentialProviderID uuid.UUID, token *UserTokenEntry) error
}

type RightSidePolicyFinder interface {
	FindGatewayPolicies(ctx context.Context, tenantID uuid.UUID, gatewayURL string) ([]RightSidePolicy, error)
	FindPolicyForServer(ctx context.Context, tenantID uuid.UUID, gatewayURL string, serverHost string, serverPort int) (*RightSidePolicy, error)
}

type RightSidePolicy struct {
	PolicyID                  uuid.UUID
	PolicyName                string
	ResourceSetID             uuid.UUID
	ServerWorkload            ServerInfo
	CredentialProviderID      uuid.UUID
	CredentialType            string
	CredentialName            string
	CredentialConfigEncrypted []byte
}

type TokenRefresher interface {
	Refresh(ctx context.Context, scope TenantScope, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error)
}

type CredentialDecrypter interface {
	DecryptCredential(configEncrypted []byte, credType string) (accessToken string, err error)
}

type RightSideAuth struct {
	policyFinder RightSidePolicyFinder
	tokenStore   UserTokenStore
	refresher    TokenRefresher
	decrypter    CredentialDecrypter
	sharedOAuth  SharedOAuthTokenResolver
	logger       *slog.Logger
}

func NewRightSideAuth(policyFinder RightSidePolicyFinder, tokenStore UserTokenStore, refresher TokenRefresher, decrypter CredentialDecrypter, sharedOAuth SharedOAuthTokenResolver) *RightSideAuth {
	return &RightSideAuth{
		policyFinder: policyFinder,
		tokenStore:   tokenStore,
		refresher:    refresher,
		decrypter:    decrypter,
		sharedOAuth:  sharedOAuth,
	}
}

func (a *RightSideAuth) SetLogger(l *slog.Logger) { a.logger = l }

func (a *RightSideAuth) GetCredential(ctx context.Context, userID string, scope TenantScope, serverHost string, serverPort int) (*ServerCredential, error) {
	if !scope.Valid() {
		return nil, ErrInvalidTenantScope
	}

	policy, err := a.policyFinder.FindPolicyForServer(ctx, scope.TenantID, scope.GatewayURL, serverHost, serverPort)
	if err != nil || policy == nil {
		return nil, ErrNoPolicyForServer
	}

	token, err := a.resolveCredential(ctx, userID, scope, policy)
	if err != nil {
		return nil, err
	}

	return &ServerCredential{
		Authorization:        token,
		ServerWorkload:       policy.ServerWorkload,
		ResourceSetID:        policy.ResourceSetID,
		PolicyID:             policy.PolicyID,
		PolicyName:           policy.PolicyName,
		CredentialType:       policy.CredentialType,
		CredentialName:       policy.CredentialName,
		CredentialProviderID: policy.CredentialProviderID,
	}, nil
}

func (a *RightSideAuth) GetAllServerCredentials(ctx context.Context, userID string, scope TenantScope) ([]ServerCredential, error) {
	if !scope.Valid() {
		return nil, ErrInvalidTenantScope
	}

	policies, err := a.policyFinder.FindGatewayPolicies(ctx, scope.TenantID, scope.GatewayURL)
	if err != nil {
		return nil, err
	}

	if len(policies) == 0 && a.logger != nil {
		a.logger.Warn("no gateway policies found",
			"tenant_id", scope.TenantID,
			"gateway_url", scope.GatewayURL,
			"user", userID,
		)
	}

	var creds []ServerCredential
	for _, policy := range policies {
		token, err := a.resolveCredential(ctx, userID, scope, &policy)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("credential resolution failed for server",
					"server", policy.ServerWorkload.Name,
					"server_id", policy.ServerWorkload.ID,
					"credential_type", policy.CredentialType,
					"user", userID,
					"error", err,
				)
			}
			continue
		}
		creds = append(creds, ServerCredential{
			Authorization:        token,
			ServerWorkload:       policy.ServerWorkload,
			ResourceSetID:        policy.ResourceSetID,
			PolicyID:             policy.PolicyID,
			PolicyName:           policy.PolicyName,
			CredentialType:       policy.CredentialType,
			CredentialName:       policy.CredentialName,
			CredentialProviderID: policy.CredentialProviderID,
		})
	}
	return creds, nil
}

func (a *RightSideAuth) GetCredentialByServerName(ctx context.Context, userID string, scope TenantScope, serverName string) (*ServerCredential, error) {
	if !scope.Valid() {
		return nil, ErrInvalidTenantScope
	}

	policies, err := a.policyFinder.FindGatewayPolicies(ctx, scope.TenantID, scope.GatewayURL)
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if toolPrefix(policy.ServerWorkload) != serverName {
			continue
		}
		token, err := a.resolveCredential(ctx, userID, scope, &policy)
		if err != nil {
			return nil, err
		}
		return &ServerCredential{
			Authorization:        token,
			ServerWorkload:       policy.ServerWorkload,
			ResourceSetID:        policy.ResourceSetID,
			PolicyID:             policy.PolicyID,
			PolicyName:           policy.PolicyName,
			CredentialType:       policy.CredentialType,
			CredentialName:       policy.CredentialName,
			CredentialProviderID: policy.CredentialProviderID,
		}, nil
	}
	return nil, ErrNoPolicyForServer
}

func (a *RightSideAuth) resolveCredential(ctx context.Context, userID string, scope TenantScope, policy *RightSidePolicy) (string, error) {
	switch policy.CredentialType {
	case "api_key", "username_password":
		if a.decrypter == nil {
			return "", ErrCredentialProviderNotConfigured
		}
		return a.decrypter.DecryptCredential(policy.CredentialConfigEncrypted, policy.CredentialType)
	case "mcp_user_based_access_token":
		token, err := a.resolvePerUserToken(ctx, userID, scope, policy)
		if err != nil {
			return "", err
		}
		return "Bearer " + token, nil
	case "oauth_authorization_code":
		if a.sharedOAuth == nil {
			return "", ErrCredentialProviderNotConfigured
		}
		token, err := a.sharedOAuth.Resolve(ctx, policy.CredentialConfigEncrypted)
		if err != nil {
			return "", err
		}
		return "Bearer " + token, nil
	default:
		return "", ErrNoPolicyForServer
	}
}

func (a *RightSideAuth) resolvePerUserToken(ctx context.Context, userID string, scope TenantScope, policy *RightSidePolicy) (string, error) {
	entry, err := a.tokenStore.Get(ctx, scope.TenantID, userID, policy.ServerWorkload.ID)
	if err != nil || entry == nil {
		return "", &ConsentRequiredError{ServerWorkloadID: policy.ServerWorkload.ID}
	}

	if time.Now().UTC().Before(entry.ExpiresAt) {
		return entry.AccessToken, nil
	}

	if a.refresher == nil || entry.RefreshToken == "" {
		return "", ErrCredentialExpired
	}

	refreshed, err := a.refresher.Refresh(ctx, scope, entry.RefreshToken, policy.ServerWorkload.ID)
	if err != nil {
		return "", ErrCredentialExpired
	}

	if storeErr := a.tokenStore.Store(ctx, scope.TenantID, userID, policy.ServerWorkload.ID, policy.CredentialProviderID, refreshed); storeErr != nil {
		return "", ErrCredentialExpired
	}

	return refreshed.AccessToken, nil
}
