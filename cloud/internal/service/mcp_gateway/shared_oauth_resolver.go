// shared_oauth_resolver.go implements AESSharedOAuthTokenResolver, which resolves the shared admin-authorized token for the OAuth 2.0 Authorization Code Credential Provider type.
package mcp_gateway

import (
	"context"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
	"github.com/dipolehq/dipole/cloud/internal/service/credentials"
)

type SharedOAuthTokenResolver interface {
	Resolve(ctx context.Context, configEncrypted []byte) (accessToken string, err error)
}

type AESSharedOAuthTokenResolver struct {
	key     []byte
	handler *credentials.OAuthAuthCodeHandler
}

func NewAESSharedOAuthTokenResolver(encryptionKey []byte) *AESSharedOAuthTokenResolver {
	return &AESSharedOAuthTokenResolver{
		key:     encryptionKey,
		handler: credentials.NewOAuthAuthCodeHandler(),
	}
}

func (r *AESSharedOAuthTokenResolver) Resolve(ctx context.Context, configEncrypted []byte) (string, error) {
	if len(configEncrypted) == 0 {
		return "", fmt.Errorf("credential config is empty")
	}

	plaintext, err := crypto.Decrypt(configEncrypted, r.key)
	if err != nil {
		return "", fmt.Errorf("decrypt credential provider config: %w", err)
	}

	cred, err := r.handler.Retrieve(ctx, plaintext, nil, nil)
	if err != nil {
		return "", err
	}

	return cred.Value, nil
}
