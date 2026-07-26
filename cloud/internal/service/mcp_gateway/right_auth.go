/*
right_auth.go implements Gateway-to-Server authorization (right-side auth).

RightSideAuth evaluates whether the Gateway can access a specific MCP server
on behalf of a user. It finds the matching right-side Access Policy (where
Client Workload = Gateway, Server Workload = MCP server), then retrieves
per-user credentials. Supports MCP User-Based Access Token (per-user OAuth)
and OAuth 2.0 Authorization Code (shared admin token).
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
)

type ServerCredential struct {
	AccessToken    string
	ServerWorkload ServerInfo
}

type ServerInfo struct {
	ID   uuid.UUID
	Name string
	Host string
	Port int
	Path string
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
	PolicyID             uuid.UUID
	ServerWorkload       ServerInfo
	CredentialProviderID uuid.UUID
	CredentialType       string
	SharedAccessToken    string
}

type TokenRefresher interface {
	Refresh(ctx context.Context, refreshToken string, serverWorkloadID uuid.UUID) (*UserTokenEntry, error)
}

type RightSideAuth struct {
	policyFinder RightSidePolicyFinder
	tokenStore   UserTokenStore
	refresher    TokenRefresher
	gatewayURL   string
}

func NewRightSideAuth(policyFinder RightSidePolicyFinder, tokenStore UserTokenStore, refresher TokenRefresher, gatewayURL string) *RightSideAuth {
	return &RightSideAuth{
		policyFinder: policyFinder,
		tokenStore:   tokenStore,
		refresher:    refresher,
		gatewayURL:   gatewayURL,
	}
}

func (a *RightSideAuth) GetCredential(ctx context.Context, userID string, tenantID uuid.UUID, serverHost string, serverPort int) (*ServerCredential, error) {
	policy, err := a.policyFinder.FindPolicyForServer(ctx, tenantID, a.gatewayURL, serverHost, serverPort)
	if err != nil || policy == nil {
		return nil, ErrNoPolicyForServer
	}

	token, err := a.resolveCredential(ctx, userID, tenantID, policy)
	if err != nil {
		return nil, err
	}

	return &ServerCredential{
		AccessToken:    token,
		ServerWorkload: policy.ServerWorkload,
	}, nil
}

func (a *RightSideAuth) GetAllServerCredentials(ctx context.Context, userID string, tenantID uuid.UUID) ([]ServerCredential, error) {
	policies, err := a.policyFinder.FindGatewayPolicies(ctx, tenantID, a.gatewayURL)
	if err != nil {
		return nil, err
	}

	var creds []ServerCredential
	for _, policy := range policies {
		token, err := a.resolveCredential(ctx, userID, tenantID, &policy)
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

func (a *RightSideAuth) GetCredentialByServerName(ctx context.Context, userID string, tenantID uuid.UUID, serverName string) (*ServerCredential, error) {
	policies, err := a.policyFinder.FindGatewayPolicies(ctx, tenantID, a.gatewayURL)
	if err != nil {
		return nil, err
	}

	for _, policy := range policies {
		if policy.ServerWorkload.Name != serverName {
			continue
		}
		token, err := a.resolveCredential(ctx, userID, tenantID, &policy)
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
	case "mcp_user_based_access_token":
		return a.resolvePerUserToken(ctx, userID, tenantID, policy)
	case "oauth_authorization_code":
		if policy.SharedAccessToken == "" {
			return "", ErrConsentRequired
		}
		return policy.SharedAccessToken, nil
	default:
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
