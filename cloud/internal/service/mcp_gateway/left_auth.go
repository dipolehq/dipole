/*
Package mcp_gateway implements the MCP Identity Gateway business logic.

left_auth.go validates inbound Bearer tokens from AI agents (left-side auth).
It verifies the JWT signature against our JWKS, checks audience matches the
Gateway URL, checks expiration, and extracts user identity from claims.
Unauthenticated requests are rejected immediately with no downstream processing.
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
)

type LeftSideAuth struct {
	issuer     *identity.IdentityIssuer
	gatewayURL string
}

type AuthResult struct {
	UserID string
	Claims map[string]any
}

func NewLeftSideAuth(issuer *identity.IdentityIssuer, gatewayURL string) *LeftSideAuth {
	return &LeftSideAuth{
		issuer:     issuer,
		gatewayURL: gatewayURL,
	}
}

func (a *LeftSideAuth) Authenticate(token string) (*AuthResult, error) {
	if token == "" {
		return nil, ErrMissingToken
	}

	claims, err := a.verifyAndDecode(token)
	if err != nil {
		return nil, err
	}

	if err := a.validateAudience(claims); err != nil {
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
		UserID: userID,
		Claims: claims,
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

func (a *LeftSideAuth) validateAudience(claims map[string]any) error {
	aud, ok := claims["aud"]
	if !ok {
		return ErrAudienceMismatch
	}

	switch v := aud.(type) {
	case string:
		if v != a.gatewayURL && v != strings.TrimSuffix(a.gatewayURL, "/") {
			return ErrAudienceMismatch
		}
	case []interface{}:
		found := false
		for _, a2 := range v {
			s, _ := a2.(string)
			if s == a.gatewayURL || s == strings.TrimSuffix(a.gatewayURL, "/") {
				found = true
				break
			}
		}
		if !found {
			return ErrAudienceMismatch
		}
	default:
		return ErrAudienceMismatch
	}

	return nil
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
