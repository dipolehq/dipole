/*
dynamic_claims.go resolves dynamic claim expressions against Trust Provider
validated token claims.

Supported pattern: ${oidc.identityToken.decode.payload.<claim_path>}
Resolves by navigating the trust claims map to extract the referenced value.
Returns the literal string unchanged if it contains no ${...} expression.
*/
package identity

import (
	"fmt"
	"strings"
)

const dynamicPrefix = "${oidc.identityToken.decode.payload."
const dynamicSuffix = "}"

func ResolveDynamicValue(value string, trustClaims map[string]interface{}) (string, error) {
	if !strings.HasPrefix(value, "${") {
		return value, nil
	}

	if !strings.HasPrefix(value, dynamicPrefix) || !strings.HasSuffix(value, dynamicSuffix) {
		return "", fmt.Errorf("unsupported dynamic expression: %s", value)
	}

	claimPath := value[len(dynamicPrefix) : len(value)-len(dynamicSuffix)]
	if claimPath == "" {
		return "", fmt.Errorf("empty claim path in expression: %s", value)
	}

	if trustClaims == nil {
		return "", fmt.Errorf("no trust claims available to resolve: %s", value)
	}

	resolved, ok := navigateClaims(trustClaims, claimPath)
	if !ok {
		return "", fmt.Errorf("claim %q not found in trust provider token", claimPath)
	}

	return resolved, nil
}

func navigateClaims(claims map[string]interface{}, path string) (string, bool) {
	if val, ok := claims[path]; ok {
		return fmt.Sprint(val), true
	}

	parts := strings.SplitN(path, ".", 2)
	if len(parts) != 2 {
		return "", false
	}

	nested, ok := claims[parts[0]]
	if !ok {
		return "", false
	}

	nestedMap, ok := nested.(map[string]interface{})
	if !ok {
		return "", false
	}

	return navigateClaims(nestedMap, parts[1])
}
