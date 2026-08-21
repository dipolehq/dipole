// aws_sts.go exchanges our OIDC token for temporary AWS credentials via STS AssumeRoleWithWebIdentity.
package credentials

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type AWSStsHandler struct {
	issuer     *identity.IdentityIssuer
	httpClient *http.Client
}

func NewAWSStsHandler(issuer *identity.IdentityIssuer) *AWSStsHandler {
	return &AWSStsHandler{
		issuer:     issuer,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type awsStsConfig struct {
	RoleARN         string `json:"role_arn"`
	SessionName     string `json:"session_name"`
	DurationSeconds int    `json:"duration_seconds"`
	Region          string `json:"region"`
}

func (h *AWSStsHandler) Retrieve(ctx context.Context, configJSON []byte, issuerURL string, evidence *domain.ClientEvidence, _ map[string]interface{}) (*domain.Credential, error) {
	var cfg awsStsConfig
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse aws sts config: %w", err)
	}

	if cfg.SessionName == "" {
		cfg.SessionName = "dipole-session"
	}
	if cfg.DurationSeconds == 0 {
		cfg.DurationSeconds = 3600
	}
	if cfg.Region == "" {
		cfg.Region = "ap-south-1"
	}

	oidcToken, err := h.issuer.IssueToken(identity.IssuanceRequest{
		IssuerURL: issuerURL,
		Subject:   evidence.DipoleClientID,
		Audience:  "sts.amazonaws.com",
		Lifetime:  5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("issue oidc token for sts: %w", err)
	}

	form := url.Values{
		"Action":           {"AssumeRoleWithWebIdentity"},
		"RoleArn":          {cfg.RoleARN},
		"RoleSessionName":  {cfg.SessionName},
		"WebIdentityToken": {oidcToken},
		"DurationSeconds":  {fmt.Sprintf("%d", cfg.DurationSeconds)},
		"Version":          {"2011-06-15"},
	}

	stsURL := fmt.Sprintf("https://sts.%s.amazonaws.com/", cfg.Region)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stsURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create sts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sts request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read sts response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sts returned %d: %s", resp.StatusCode, string(body))
	}

	var stsResp assumeRoleResponse
	if err := xml.Unmarshal(body, &stsResp); err != nil {
		return nil, fmt.Errorf("parse sts xml: %w", err)
	}

	creds := stsResp.Result.Credentials
	expiration, _ := time.Parse(time.RFC3339, creds.Expiration)

	return &domain.Credential{
		Type:               "aws_sts",
		AWSAccessKeyID:     creds.AccessKeyID,
		AWSSecretAccessKey: creds.SecretAccessKey,
		AWSSessionToken:    creds.SessionToken,
		ExpiresAt:          expiration.Unix(),
	}, nil
}

type assumeRoleResponse struct {
	XMLName xml.Name         `xml:"AssumeRoleWithWebIdentityResponse"`
	Result  assumeRoleResult `xml:"AssumeRoleWithWebIdentityResult"`
}

type assumeRoleResult struct {
	Credentials stsCredentials `xml:"Credentials"`
}

type stsCredentials struct {
	AccessKeyID     string `xml:"AccessKeyId"`
	SecretAccessKey string `xml:"SecretAccessKey"`
	SessionToken    string `xml:"SessionToken"`
	Expiration      string `xml:"Expiration"`
}
