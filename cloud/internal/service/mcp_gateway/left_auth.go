/*
Package mcp_gateway implements the MCP Identity Gateway business logic.

left_auth.go validates inbound Bearer tokens from AI agents (left-side auth).
It verifies the JWT signature against our JWKS, checks the audience, checks
expiration, and extracts user identity from claims. Unauthenticated requests are
rejected immediately with no downstream processing.

The expected audience is supplied PER CALL, not held on the struct. It is the
Gateway URL of the tenant whose hostname the request arrived on, and it is what
binds a token to one tenant: the signing key is shared platform-wide, so a token
minted for tenant A's Gateway is distinguishable from tenant B's only by its
`aud` claim. Holding one audience on the struct made every tenant's tokens get
checked against whichever tenant was configured at startup.
*/
package mcp_gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

var (
	ErrMissingToken     = errors.New("missing bearer token")
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrAudienceMismatch = errors.New("token audience does not match gateway")
	ErrNoUserIdentity   = errors.New("no user identity in token claims")
	// ErrNoExpectedAudience means the caller did not say which Gateway URL the
	// token must be addressed to. Treated as a failure rather than "accept any
	// audience", because accepting any audience would accept every other
	// tenant's tokens.
	ErrNoExpectedAudience = errors.New("expected gateway audience not supplied")
)

type LeftSideAuth struct {
	issuer *identity.IdentityIssuer
}

type AuthResult struct {
	UserID             string
	Claims             map[string]any
	ClientWorkloadName string
}

func NewLeftSideAuth(issuer *identity.IdentityIssuer) *LeftSideAuth {
	return &LeftSideAuth{issuer: issuer}
}

// Authenticate validates an inbound Bearer token from an AI agent.
//
// expectedAudience must be the Gateway URL of the tenant that owns the hostname
// this request arrived on. It is required: see ErrNoExpectedAudience.
func (a *LeftSideAuth) Authenticate(token string, expectedAudience string) (*AuthResult, error) {
	if token == "" {
		return nil, ErrMissingToken
	}
	if expectedAudience == "" {
		return nil, ErrNoExpectedAudience
	}

	claims, err := a.verifyAndDecode(token)
	if err != nil {
		return nil, err
	}

	if err := validateAudience(claims, expectedAudience); err != nil {
		return nil, err
	}

	if err := a.validateExpiry(claims); err != nil {
		return nil, err
	}

	userID := extractUserID(claims)
	if userID == "" {
		return nil, ErrNoUserIdentity
	}

	return &AuthResult{
		UserID:             userID,
		Claims:             claims,
		ClientWorkloadName: extractClientWorkloadName(claims),
	}, nil
}

func (a *LeftSideAuth) verifyAndDecode(token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	// Verify signature using our JWKS
	jwks := a.issuer.JWKS()
	keys, ok := jwks["keys"].([]map[string]interface{})
	if !ok || len(keys) == 0 {
		return nil, fmt.Errorf("no signing keys available")
	}

	// Decode header to get algorithm and kid
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, ErrInvalidToken
	}

	// Find matching key and verify
	verified := false
	for _, key := range keys {
		kid, _ := key["kid"].(string)
		alg, _ := key["alg"].(string)
		if kid == header.Kid && alg == header.Alg {
			if err := a.issuer.VerifyTokenSignature(token); err == nil {
				verified = true
				break
			}
		}
	}

	if !verified {
		// Fallback: try verification without kid matching (for tokens from our own issuer)
		if err := a.issuer.VerifyTokenSignature(token); err != nil {
			return nil, ErrInvalidToken
		}
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// validateAudience checks the token's `aud` claim against the Gateway URL of
// the tenant handling this request. `aud` may be a single string or an array
// (RFC 7519 §4.1.3); the array form matches if any element matches.
func validateAudience(claims map[string]any, expectedAudience string) error {
	aud, ok := claims["aud"]
	if !ok {
		return ErrAudienceMismatch
	}

	switch v := aud.(type) {
	case string:
		if !audienceMatches(v, expectedAudience) {
			return ErrAudienceMismatch
		}
	case []interface{}:
		for _, entry := range v {
			s, isString := entry.(string)
			if isString && audienceMatches(s, expectedAudience) {
				return nil
			}
		}
		return ErrAudienceMismatch
	default:
		return ErrAudienceMismatch
	}

	return nil
}

// audienceMatches compares two audience values, normalising a trailing slash on
// BOTH sides. The customer types the audience into their Credential Provider
// config by hand, so "https://x-mcpgateway.example.com" and the same value with
// a trailing slash have to be treated as the same audience. Normalising only the
// expected side (as this previously did) rejected tokens whose own `aud` carried
// the slash.
func audienceMatches(got, expected string) bool {
	return strings.TrimSuffix(got, "/") == strings.TrimSuffix(expected, "/")
}

func (a *LeftSideAuth) validateExpiry(claims map[string]any) error {
	exp, ok := claims["exp"]
	if !ok {
		return ErrTokenExpired
	}

	var expTime float64
	switch v := exp.(type) {
	case float64:
		expTime = v
	case json.Number:
		f, _ := v.Float64()
		expTime = f
	default:
		return ErrTokenExpired
	}

	if time.Now().Unix() > int64(expTime) {
		return ErrTokenExpired
	}

	return nil
}

func extractUserID(claims map[string]any) string {
	if email, ok := claims["email"].(string); ok && email != "" {
		return email
	}
	if sub, ok := claims["sub"].(string); ok && sub != "" {
		return sub
	}
	return ""
}

func extractClientWorkloadName(claims map[string]any) string {
	name, _ := claims["client_workload_name"].(string)
	return name
}
