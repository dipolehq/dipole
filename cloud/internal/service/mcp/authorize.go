/*
authorize.go orchestrates the OAuth 2.1 authorization flow for MCP clients.

AuthorizeService validates the authorization request, finds the matching
Access Policy (by redirect_uri → Client Workload, resource → Server Workload),
evaluates access conditions, and issues an authorization code.
If Enforce SSO is on, it returns a redirect URL to the IdP instead.
*/
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

var (
	ErrClientNotFound         = errors.New("client_id not found")
	ErrRedirectURIMismatch    = errors.New("redirect_uri does not match registered URIs")
	ErrInvalidChallengeMethod = errors.New("code_challenge_method must be S256")
	ErrMissingCodeChallenge   = errors.New("code_challenge is required")
	ErrNoPolicyFound          = errors.New("No matching Access Policy found")
	ErrConditionFailed        = errors.New("access condition failed")
)

type AuthorizeRequest struct {
	ClientID            string
	RedirectURI         string
	ResponseType        string
	CodeChallenge       string
	CodeChallengeMethod string
	Scope               string
	State               string
	Resource            string
}

type AuthorizeResult struct {
	RedirectURL string
	IdPRedirect string
}

type PolicyFinder interface {
	FindMCPPolicy(ctx context.Context, tenantID uuid.UUID, redirectURI string, host string, port int) (*domain.AccessPolicyWithRelations, error)
}

type ConditionChecker interface {
	EvaluateAll(ctx context.Context, conditions []domain.AccessCondition, evidence *domain.ClientEvidence) (bool, string, error)
}

type AuthorizeService struct {
	dcr        *DCRService
	codeStore  *AuthCodeStore
	policies   PolicyFinder
	conditions ConditionChecker
}

func NewAuthorizeService(dcr *DCRService, codeStore *AuthCodeStore, policies PolicyFinder, conditions ConditionChecker) *AuthorizeService {
	return &AuthorizeService{
		dcr:        dcr,
		codeStore:  codeStore,
		policies:   policies,
		conditions: conditions,
	}
}

func (s *AuthorizeService) Authorize(ctx context.Context, tenantID uuid.UUID, req *AuthorizeRequest, clientIP string) (*AuthorizeResult, error) {
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, ErrClientNotFound
	}

	client, err := s.dcr.GetClient(ctx, clientUUID)
	if err != nil {
		return nil, ErrClientNotFound
	}

	if !MatchRedirectURIInList(req.RedirectURI, client.RedirectURIs) {
		return nil, ErrRedirectURIMismatch
	}

	if req.CodeChallengeMethod != "S256" {
		return nil, ErrInvalidChallengeMethod
	}
	if req.CodeChallenge == "" {
		return nil, ErrMissingCodeChallenge
	}

	host, port, err := parseResource(req.Resource)
	if err != nil {
		return nil, fmt.Errorf("invalid resource parameter: %w", err)
	}

	policy, err := s.policies.FindMCPPolicy(ctx, tenantID, req.RedirectURI, host, port)
	if err != nil || policy == nil {
		return nil, ErrNoPolicyFound
	}

	// TODO (Task 5): If Enforce SSO = ON, redirect to IdP here and return IdPRedirect.
	// For now, we handle the SSO-off path: skip user auth, evaluate conditions, issue code.

	if len(policy.AccessConditions) > 0 && s.conditions != nil {
		evidence := &domain.ClientEvidence{SourceIP: clientIP}
		passed, reason, err := s.conditions.EvaluateAll(ctx, policy.AccessConditions, evidence)
		if err != nil {
			return nil, err
		}
		if !passed {
			return nil, fmt.Errorf("%w: %s", ErrConditionFailed, reason)
		}
	}

	scope := req.Scope
	if scope == "" {
		scope = "openid"
	}

	code := s.codeStore.Create(
		client.ID,
		tenantID,
		req.RedirectURI,
		req.CodeChallenge,
		scope,
		nil,
		policy.Policy.ID,
	)

	redirectURL := buildAuthCodeRedirect(req.RedirectURI, code, req.State)
	return &AuthorizeResult{RedirectURL: redirectURL}, nil
}

func buildAuthCodeRedirect(redirectURI, code, state string) string {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func parseResource(resource string) (string, int, error) {
	if resource == "" {
		return "", 0, errors.New("resource parameter is required")
	}

	u, err := url.Parse(resource)
	if err != nil {
		return "", 0, err
	}

	host := u.Hostname()
	if host == "" {
		return "", 0, errors.New("resource must contain a host")
	}

	port := 443
	if u.Scheme == "http" {
		port = 80
	}
	if u.Port() != "" {
		p, err := strconv.Atoi(u.Port())
		if err != nil {
			return "", 0, fmt.Errorf("invalid port: %w", err)
		}
		port = p
	}

	return host, port, nil
}
