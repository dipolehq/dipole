// jwt_verify.go centralizes JWS parsing, key-set resolution, signature verification, and claims decoding shared by every JWKS-based validator.
package trust

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-jose/go-jose/v4"
)

type keySetResolver interface {
	Resolve(ctx context.Context) (*jose.JSONWebKeySet, error)
}

type cachedKeySetResolver struct {
	cache    *JWKSCache
	endpoint string
}

func (r cachedKeySetResolver) Resolve(ctx context.Context) (*jose.JSONWebKeySet, error) {
	return r.cache.Get(ctx, r.endpoint)
}

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

type verifyStage int

const (
	stageParse verifyStage = iota
	stageResolveKeySet
	stageKeyNotFound
	stageVerify
	stageDecodeClaims
)

type jwsVerifyError struct {
	stage verifyStage
	cause error
}

func (e *jwsVerifyError) Error() string { return e.cause.Error() }

func (e *jwsVerifyError) keySetUnavailable() bool {
	return e.stage == stageResolveKeySet
}
