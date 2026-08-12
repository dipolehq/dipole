/*
right_auth.go implements Gateway-to-Server authorization (right-side auth).

RightSideAuth evaluates whether the Gateway can access a specific MCP server
on behalf of a user. It finds the matching right-side Access Policy (where
Client Workload = Gateway, Server Workload = MCP server), then retrieves
per-user credentials. Supports MCP User-Based Access Token (per-user OAuth)
and OAuth 2.0 Authorization Code (shared admin token).

The tenant arrives per call as a TenantScope, never held on the struct: one
process serves every tenant's Gateway, and a captured tenant would scope every
tenant's policy lookups to whichever one was configured at startup. Calls with
an invalid scope are refused before the policy finder is consulted.
*/
package mcp_gateway

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNoPolicyForServer = errors.New("no access policy found for this MCP server")
	ErrConsentRequired   = errors.New("user authorization required for this MCP server")
	ErrCredentialExpired = errors.New("credential expired and refresh failed")
	// ErrInvalidTenantScope means the request was not attributed to a tenant.
	// Fail closed: continuing would query policies across no tenant, or match a
	// Client Workload whose audience identifier happens to be empty.
	ErrInvalidTenantScope = errors.New("request has no valid tenant scope")
)

type ServerCredential struct {
	AccessToken    string
	ServerWorkload ServerInfo
}

// ServerInfo describes a downstream MCP server, carrying everything needed both
// to identify it in a policy and to open a connection to it. The connection
// details travel with the policy result on purpose: the Gateway must only ever
// talk to a server that an Access Policy authorised for this tenant, so deriving
// the endpoint from the policy makes reaching an unauthorised server impossible
// by construction.
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
	Store(ctx context.Context, tenantID uuid.UUID, userID string, serverWorkloadID uuid.UUID, token *UserTokenEntry) error
}

type RightSidePolicyFinder interface {
	FindGatewayPolicies(ctx context.Context, tenantID uuid.UUID, gatewayURL string) ([]RightSidePolicy, error)
	FindPolicyForServer(ctx context.Context, tenantID uuid.UUID, gatewayURL string, serverHost string, serverPort int) (*RightSidePolicy, error)
}

type RightSidePolicy struct {
	PolicyID                  uuid.UUID
	ServerWorkload            ServerInfo
	CredentialProviderID      uuid.UUID
	CredentialType            string
	CredentialConfigEncrypted []byte
	SharedAccessToken         string
}

type TokenRefresher interface {
	Refresh(ctx context.Context, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error)
}

// CredentialDecrypter decrypts a credential provider's encrypted config and
// extracts the usable token/key. Injected so right_auth.go has no dependency
// on the crypto package or encryption key directly.
type CredentialDecrypter interface {
	DecryptCredential(configEncrypted []byte, credType string) (accessToken string, err error)
}

// RightSideAuth holds no tenant state. The tenant and its Gateway URL arrive
// with each call as a TenantScope, resolved from the request hostname.
type RightSideAuth struct {
	policyFinder RightSidePolicyFinder
	tokenStore   UserTokenStore
	refresher    TokenRefresher
	decrypter    CredentialDecrypter
}

func NewRightSideAuth(policyFinder RightSidePolicyFinder, tokenStore UserTokenStore, refresher TokenRefresher, decrypter CredentialDecrypter) *RightSideAuth {
	return &RightSideAuth{
		policyFinder: policyFinder,
		tokenStore:   tokenStore,
		refresher:    refresher,
		decrypter:    decrypter,
	}
}

func (a *RightSideAuth) GetCredential(ctx context.Context, userID string, scope TenantScope, serverHost string, serverPort int) (*ServerCredential, error) {
	if !scope.Valid() {
		return nil, ErrInvalidTenantScope
	}

	policy, err := a.policyFinder.FindPolicyForServer(ctx, scope.TenantID, scope.GatewayURL, serverHost, serverPort)
	if err != nil || policy == nil {
		return nil, ErrNoPolicyForServer
	}

	token, err := a.resolveCredential(ctx, userID, scope.TenantID, policy)
	if err != nil {
		return nil, err
	}

	return &ServerCredential{
		AccessToken:    token,
		ServerWorkload: policy.ServerWorkload,
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

	var creds []ServerCredential
	for _, policy := range policies {
		token, err := a.resolveCredential(ctx, userID, scope.TenantID, &policy)
		if err != nil {
			continue
		}
		creds = append(creds, ServerCredential{
			AccessToken:    token,
			ServerWorkload: policy.ServerWorkload,
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
		token, err := a.resolveCredential(ctx, userID, scope.TenantID, &policy)
		if err != nil {
			return nil, err
		}
		return &ServerCredential{
			AccessToken:    token,
			ServerWorkload: policy.ServerWorkload,
		}, nil
	}
	return nil, ErrNoPolicyForServer
}

func (a *RightSideAuth) resolveCredential(ctx context.Context, userID string, tenantID uuid.UUID, policy *RightSidePolicy) (string, error) {
	switch policy.CredentialType {
	case "api_key", "username_password":
		if a.decrypter == nil {
			return "", ErrNoPolicyForServer
		}
		return a.decrypter.DecryptCredential(policy.CredentialConfigEncrypted, policy.CredentialType)
	case "mcp_user_based_access_token":
		return a.resolvePerUserToken(ctx, userID, tenantID, policy)
	case "oauth_authorization_code":
		if policy.SharedAccessToken == "" {
			return "", ErrConsentRequired
		}
		return policy.SharedAccessToken, nil
	default:
		// Unsupported credential type for Gateway right-side auth.
		return "", ErrNoPolicyForServer
	}
}

func (a *RightSideAuth) resolvePerUserToken(ctx context.Context, userID string, tenantID uuid.UUID, policy *RightSidePolicy) (string, error) {
	entry, err := a.tokenStore.Get(ctx, tenantID, userID, policy.ServerWorkload.ID)
	if err != nil || entry == nil {
		return "", ErrConsentRequired
	}

	if time.Now().UTC().Before(entry.ExpiresAt) {
		return entry.AccessToken, nil
	}

	if a.refresher == nil || entry.RefreshToken == "" {
		return "", ErrCredentialExpired
	}

	refreshed, err := a.refresher.Refresh(ctx, entry.RefreshToken, policy.ServerWorkload.ID)
	if err != nil {
		return "", ErrCredentialExpired
	}

	if storeErr := a.tokenStore.Store(ctx, tenantID, userID, policy.ServerWorkload.ID, refreshed); storeErr != nil {
		return "", ErrCredentialExpired
	}

	return refreshed.AccessToken, nil
}
