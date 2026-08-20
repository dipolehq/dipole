/*
jwt_verify.go centralizes the "parse a compact JWS, fetch its signing key
from a JWKS source, verify the signature, and decode claims" mechanism
shared by every JWKS-based validator (K8s, GitHub, OIDC's oidc_discovery
and upload_jwks paths, GCP).

Before this refactor, four validators each implemented this sequence
independently. That duplication is exactly what let the Part 1 re-review
catch a real bug: the issuer trailing-slash fix had to be applied to two
files by hand, and it would have been trivial to miss a third. Routing
every JWKS-based validator through verifyJWS closes that class of bug
going forward — a fix or a new check written once here applies everywhere
it's used, with no way for one caller to silently diverge from another.

This file intentionally does NOT unify claim-level policy checks (issuer,
audience, exp, nbf) across validators — those differ deliberately by
design (see oidc_validator.go's comment on why upload_jwks/symmetric_key
skip the issuer check, and gcp_validator.go's currently-narrower check
set, which is a tracked, separate gap, not something to paper over here).
Only the mechanical signature-verification steps that are byte-for-byte
identical across validators are centralized.
*/
package trust

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v4"
)

// keySetResolver returns the JSON Web Key Set to verify a JWT against.
// K8s/OIDC-discovery/GCP resolve this via the shared JWKSCache (keyed by
// a customer-configured or well-known OIDC endpoint); OIDC's upload_jwks
// path resolves it from a JSON blob embedded directly in the Trust
// Provider's config instead of any network fetch. Abstracting the source
// behind this one-method interface is what lets both paths share the
// same parse/verify/decode mechanism below.
type keySetResolver interface {
	Resolve(ctx context.Context) (*jose.JSONWebKeySet, error)
}

// cachedKeySetResolver resolves via the shared JWKSCache, keyed by
// endpoint. This is the resolver K8sValidator, GitHubValidator,
// OIDCValidator's oidc_discovery method, and GCPValidator all use.
type cachedKeySetResolver struct {
	cache    *JWKSCache
	endpoint string
}

func (r cachedKeySetResolver) Resolve(ctx context.Context) (*jose.JSONWebKeySet, error) {
	return r.cache.Get(ctx, r.endpoint)
}

// staticKeySetResolver resolves from an already-known JWKS document,
// used by OIDCValidator's upload_jwks method where the key set is
// embedded in Trust Provider config rather than fetched over the network.
type staticKeySetResolver struct {
	jwksJSON string
}

func (r staticKeySetResolver) Resolve(_ context.Context) (*jose.JSONWebKeySet, error) {
	var keySet jose.JSONWebKeySet
	if err := json.Unmarshal([]byte(r.jwksJSON), &keySet); err != nil {
		return nil, fmt.Errorf("parse uploaded JWKS: %w", err)
	}
	return &keySet, nil
}

// verifyJWS parses token as a compact-serialized JWS restricted to
// RS256/ES256 (the two algorithms every JWKS-based validator in this
// package has always required — go-jose's ParseSigned enforces this
// allowlist at parse time, so a token using any other algorithm, notably
// the symmetric HS256 family, is rejected before a key lookup is even
// attempted), resolves the signing key set via resolver, finds the key
// matching the token's kid, verifies the signature, and decodes the
// payload as a claims map.
//
// verifyJWS returns three distinct error categories via the returned
// jwsVerifyError, because existing callers currently differ in how
// finely they report failure: K8sValidator/GitHubValidator distinguish
// "couldn't fetch the JWKS" (Reason: NoDataFound) from every other
// failure (Reason: InvalidSignature), while OIDCValidator/GCPValidator
// collapse everything to InvalidSignature. Returning a typed error lets
// each caller keep its own existing Reason mapping unchanged — this
// refactor centralizes the mechanism, not each validator's response
// vocabulary, so there is zero externally observable behavior change.
func verifyJWS(ctx context.Context, token string, resolver keySetResolver) (map[string]interface{}, *jwsVerifyError) {
	jws, err := jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256, jose.ES256})
	if err != nil {
		return nil, &jwsVerifyError{stage: stageParse, cause: fmt.Errorf("parse JWS: %w", err)}
	}

	keySet, err := resolver.Resolve(ctx)
	if err != nil {
		return nil, &jwsVerifyError{stage: stageResolveKeySet, cause: fmt.Errorf("resolve key set: %w", err)}
	}

	kid := jws.Signatures[0].Header.KeyID
	keys := keySet.Key(kid)
	if len(keys) == 0 {
		return nil, &jwsVerifyError{stage: stageKeyNotFound, cause: fmt.Errorf("no key in JWKS matches kid %q", kid)}
	}

	payload, err := jws.Verify(keys[0])
	if err != nil {
		return nil, &jwsVerifyError{stage: stageVerify, cause: fmt.Errorf("verify signature: %w", err)}
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, &jwsVerifyError{stage: stageDecodeClaims, cause: fmt.Errorf("decode claims: %w", err)}
	}
	return claims, nil
}

// verifyStage identifies which step of verifyJWS failed.
type verifyStage int

const (
	stageParse verifyStage = iota
	stageResolveKeySet
	stageKeyNotFound
	stageVerify
	stageDecodeClaims
)

// jwsVerifyError carries both the underlying cause (for logging) and
// which stage failed (for callers that map specific stages to specific
// Reason strings, e.g. "the JWKS was unreachable" vs. "the signature was
// bad").
type jwsVerifyError struct {
	stage verifyStage
	cause error
}

func (e *jwsVerifyError) Error() string { return e.cause.Error() }

// keySetUnavailable reports whether the failure happened while resolving
// the key set (network/config problem) rather than while validating the
// token itself. Callers that want a distinct "couldn't even check this"
// reason (K8sValidator, GitHubValidator both map this to NoDataFound)
// use this; callers that don't care collapse every jwsVerifyError to one
// Reason regardless.
func (e *jwsVerifyError) keySetUnavailable() bool {
	return e.stage == stageResolveKeySet
}
