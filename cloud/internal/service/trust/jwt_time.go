/*
jwt_time.go provides shared time-based claim validation (exp, nbf) used by
every JWT-based validator (K8s, GitHub, OIDC). Centralized here so the skew
tolerance and comparison logic can't drift out of sync between validators —
exactly the kind of duplication that previously let one validator diverge
from another.
*/
package trust

import "time"

// clockSkew tolerates minor clock drift between the token issuer and this
// server when checking time-bound claims. Matches the tolerance already
// established by samlClockSkew in saml_validator.go for the same reason:
// without it, a token whose nbf is "now" at the issuer, or whose exp has
// just passed, can be spuriously rejected if this server's clock runs
// even slightly ahead or behind — an intermittent, hard-to-diagnose
// failure for customers rather than a security concern.
const clockSkew = 180 * time.Second

// nbfNotYetValid reports whether claims["nbf"] is a future time (per RFC
// 7519 §4.1.5), beyond clockSkew tolerance. Returns false (not rejected)
// if nbf is absent or not a number — an absent nbf places no additional
// restriction on the token per the RFC, it simply means the claim wasn't
// asserted.
//
// Note: the pre-existing exp (expiry) check in each validator does not
// currently apply this same tolerance. That's intentionally left as-is
// here since it predates this change and isn't part of it — tightening
// or loosening exp handling is a separate decision with its own tradeoffs
// (a skew-tolerant exp check extends how long a token remains usable past
// its stated expiry).
func nbfNotYetValid(claims map[string]interface{}) bool {
	nbf, ok := claims["nbf"].(float64)
	if !ok {
		return false
	}
	return time.Unix(int64(nbf), 0).After(time.Now().Add(clockSkew))
}
