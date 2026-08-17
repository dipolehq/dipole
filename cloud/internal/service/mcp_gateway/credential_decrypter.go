// credential_decrypter.go implements AESCredentialDecrypter, which decrypts API Key and Username/Password credential configs into ready-to-use Authorization header values.
package mcp_gateway

import (
	"encoding/base64"
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
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return "", fmt.Errorf("parse api key config: %w", err)
	}
	if config.APIKey == "" {
		return "", fmt.Errorf("api key is empty in credential config")
	}
	return "Bearer " + config.APIKey, nil
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
	encoded := base64.StdEncoding.EncodeToString([]byte(config.Username + ":" + config.Password))
	return "Basic " + encoded, nil
}
