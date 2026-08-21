// verify_id_token.go provides VerifyIDToken, a high-level helper for verifying OIDC id_tokens via JWKS with issuer, audience, and expiry checks.
package trust

import (
	"context"
	"fmt"
	"strings"
)

func VerifyIDToken(ctx context.Context, token string, cache *JWKSCache, expectedIssuer, expectedAudience string) (map[string]interface{}, error) {
	if cache == nil {
		return nil, fmt.Errorf("JWKS cache is nil")
	}
	if expectedIssuer == "" {
		return nil, fmt.Errorf("expected issuer must not be empty")
	}

	claims, verr := verifyJWS(ctx, token, cachedKeySetResolver{cache: cache, endpoint: expectedIssuer})
	if verr != nil {
		return nil, fmt.Errorf("signature verification failed: %w", verr)
	}

	iss, _ := claims["iss"].(string)
	if strings.TrimRight(iss, "/") != strings.TrimRight(expectedIssuer, "/") {
		return nil, fmt.Errorf("issuer mismatch: got %q, want %q", iss, expectedIssuer)
	}

	if expectedAudience != "" {
		if !audienceMatches(claims, expectedAudience) {
			return nil, fmt.Errorf("audience mismatch: token does not contain %q", expectedAudience)
		}
	}

	if expIsExpired(claims) {
		return nil, fmt.Errorf("id_token is expired")
	}

	if nbfNotYetValid(claims) {
		return nil, fmt.Errorf("id_token is not yet valid")
	}

	return claims, nil
}

func audienceMatches(claims map[string]interface{}, expected string) bool {
	switch aud := claims["aud"].(type) {
	case string:
		return aud == expected
	case []interface{}:
		for _, v := range aud {
			if s, ok := v.(string); ok && s == expected {
				return true
			}
		}
	}
	return false
}
