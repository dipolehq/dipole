// aws_secrets_manager.go retrieves secrets from AWS Secrets Manager by assuming an IAM role via STS, then calling GetSecretValue with SigV4 signing.
package credentials

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type AWSSecretsManagerHandler struct {
	issuer     *identity.IdentityIssuer
	stsHandler *AWSStsHandler
	httpClient *http.Client
}

func NewAWSSecretsManagerHandler(issuer *identity.IdentityIssuer) *AWSSecretsManagerHandler {
	return &AWSSecretsManagerHandler{
		issuer:     issuer,
		stsHandler: NewAWSStsHandler(issuer),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type awsSecretsManagerConfig struct {
	SecretARN string `json:"secret_arn"`
	Region    string `json:"region"`
	RoleARN   string `json:"role_arn"`
}

func (h *AWSSecretsManagerHandler) Retrieve(ctx context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg awsSecretsManagerConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse aws secrets manager config: %w", err)
	}

	if cfg.Region == "" {
		cfg.Region = "ap-south-1"
	}

	stsConfig, _ := json.Marshal(awsStsConfig{
		RoleARN:         cfg.RoleARN,
		SessionName:     "dipole-secrets-session",
		DurationSeconds: 900,
		Region:          cfg.Region,
	})

	stsCred, err := h.stsHandler.Retrieve(ctx, stsConfig, issuerURL, evidence, nil)
	if err != nil {
		return nil, fmt.Errorf("assume role for secrets manager: %w", err)
	}

	secretValue, err := h.getSecretValue(ctx, cfg.SecretARN, cfg.Region, stsCred)
	if err != nil {
		return nil, fmt.Errorf("get secret value: %w", err)
	}

	return &domain.Credential{Type: "bearer", Value: secretValue}, nil
}

func (h *AWSSecretsManagerHandler) getSecretValue(ctx context.Context, secretARN, region string, creds *domain.Credential) (string, error) {
	endpoint := fmt.Sprintf("https://secretsmanager.%s.amazonaws.com", region)
	body := fmt.Sprintf(`{"SecretId":"%s"}`, secretARN)

	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-amz-json-1.1")
	req.Header.Set("X-Amz-Target", "secretsmanager.GetSecretValue")
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Security-Token", creds.AWSSessionToken)

	signedHeaders := "content-type;host;x-amz-date;x-amz-security-token;x-amz-target"
	host := fmt.Sprintf("secretsmanager.%s.amazonaws.com", region)

	canonicalHeaders := "content-type:application/x-amz-json-1.1\n" +
		"host:" + host + "\n" +
		"x-amz-date:" + amzDate + "\n" +
		"x-amz-security-token:" + creds.AWSSessionToken + "\n" +
		"x-amz-target:secretsmanager.GetSecretValue\n"

	bodyHash := sha256Hex([]byte(body))
	canonicalRequest := "POST\n/\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + bodyHash

	credentialScope := dateStamp + "/" + region + "/secretsmanager/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + credentialScope + "\n" + sha256Hex([]byte(canonicalRequest))

	signingKey := deriveKey(creds.AWSSecretAccessKey, dateStamp, region, "secretsmanager")
	signature := hex.EncodeToString(hmacSHA(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.AWSAccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("secrets manager returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		SecretString string `json:"SecretString"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", err
	}

	return result.SecretString, nil
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func deriveKey(secret, dateStamp, region, service string) []byte {
	kDate := hmacSHA([]byte("AWS4"+secret), []byte(dateStamp))
	kRegion := hmacSHA(kDate, []byte(region))
	kService := hmacSHA(kRegion, []byte(service))
	return hmacSHA(kService, []byte("aws4_request"))
}
