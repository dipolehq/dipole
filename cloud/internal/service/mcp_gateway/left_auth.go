// left_auth.go implements LeftSideAuth, which validates inbound Bearer tokens from AI agents against the Gateway's expected audience.
package mcp_gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

var (
	ErrMissingToken       = errors.New("missing bearer token")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrAudienceMismatch   = errors.New("token audience does not match gateway")
	ErrNoUserIdentity     = errors.New("no user identity in token claims")
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

	if err := a.issuer.VerifyTokenSignature(token); err != nil {
		return nil, ErrInvalidToken
	}

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
