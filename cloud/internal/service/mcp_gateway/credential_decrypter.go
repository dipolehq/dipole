/*
credential_decrypter.go decrypts simple credential provider configs (API Key,
Username/Password) for right-side auth injection. Keeps crypto concerns out of
the auth logic itself.
*/
package mcp_gateway

import (
	"encoding/json"
	"fmt"

	"github.com/dipolehq/dipole/cloud/internal/crypto"
)

type AESCredentialDecrypter struct {
	key []byte
}

func NewAESCredentialDecrypter(encryptionKey []byte) *AESCredentialDecrypter {
	return &AESCredentialDecrypter{key: encryptionKey}
}

func (d *AESCredentialDecrypter) DecryptCredential(configEncrypted []byte, credType string) (string, error) {
	if len(configEncrypted) == 0 {
		return "", fmt.Errorf("credential config is empty")
	}

	plaintext, err := crypto.Decrypt(configEncrypted, d.key)
	if err != nil {
		return "", fmt.Errorf("decrypt credential config: %w", err)
	}

	switch credType {
	case "api_key":
		return extractAPIKey(plaintext)
	case "username_password":
		return extractUsernamePassword(plaintext)
	default:
		return "", fmt.Errorf("unsupported simple credential type: %s", credType)
	}
}

func extractAPIKey(configJSON []byte) (string, error) {
	var config struct {
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return "", fmt.Errorf("parse api key config: %w", err)
	}
	if config.APIKey == "" {
		return "", fmt.Errorf("api key is empty in credential config")
	}
	return config.APIKey, nil
}

func extractUsernamePassword(configJSON []byte) (string, error) {
	var config struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return "", fmt.Errorf("parse username/password config: %w", err)
	}
	if config.Password == "" {
		return "", fmt.Errorf("password is empty in credential config")
	}
	// For Bearer injection, use password as the token value.
	// The server workload's auth scheme determines actual injection format.
	return config.Password, nil
}
