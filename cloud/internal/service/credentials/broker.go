/*
Package credentials implements the Credential Broker that retrieves credentials
from Credential Providers.

CredentialBrokerImpl decrypts stored credential configs (AES-256-GCM) and
returns the credential values. Supports API Key and Username/Password for now.
Decrypted values are never logged, cached, or stored beyond the function call.
*/
package credentials

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository/postgres"
)

type CredentialBroker interface {
	Retrieve(ctx context.Context, provider *domain.CredentialProvider, evidence *domain.ClientEvidence) (*domain.Credential, error)
}

type CredentialBrokerImpl struct {
	encryptionKey []byte
}

func NewCredentialBroker(encryptionKey []byte) *CredentialBrokerImpl {
	return &CredentialBrokerImpl{encryptionKey: encryptionKey}
}

func (b *CredentialBrokerImpl) Retrieve(_ context.Context, provider *domain.CredentialProvider, _ *domain.ClientEvidence) (*domain.Credential, error) {
	switch provider.Type {
	case "api_key":
		return b.retrieveAPIKey(provider.ConfigEncrypted)
	case "username_password":
		return b.retrieveUsernamePassword(provider.ConfigEncrypted)
	default:
		return nil, fmt.Errorf("credential provider type %q not implemented", provider.Type)
	}
}

func (b *CredentialBrokerImpl) retrieveAPIKey(encrypted []byte) (*domain.Credential, error) {
	plaintext, err := postgres.Decrypt(encrypted, b.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt api key config: %w", err)
	}

	var config struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("parse api key config: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: config.APIKey}, nil
}

func (b *CredentialBrokerImpl) retrieveUsernamePassword(encrypted []byte) (*domain.Credential, error) {
	plaintext, err := postgres.Decrypt(encrypted, b.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt username/password config: %w", err)
	}

	var config struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(plaintext, &config); err != nil {
		return nil, fmt.Errorf("parse username/password config: %w", err)
	}

	return &domain.Credential{Type: "basic", Username: config.Username, Password: config.Password}, nil
}
