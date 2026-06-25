/*
aws_role_validator.go verifies AWS IAM Role identity by replaying a SigV4-signed
STS GetCallerIdentity request.

AWSRoleValidator reconstructs an HTTP POST to STS using the signed headers from
evidence. If STS accepts the request (HTTP 200), the identity is proven. The
response ARN, Account, and UserId are then evaluated against match rules.
This is the only validator that makes an outbound HTTP call from Cloud.
*/
package trust

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type AWSRoleValidator struct {
	httpClient *http.Client
}

func NewAWSRoleValidator() *AWSRoleValidator {
	return &AWSRoleValidator{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

type awsRoleConfig struct {
	MatchRules []MatchRule `json:"match_rules"`
}

func (v *AWSRoleValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.AWS == nil || evidence.AWS.STSGetCallerIdentity == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	sts := evidence.AWS.STSGetCallerIdentity
	if len(sts.SignedHeaders) == 0 || sts.Region == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	var cfg awsRoleConfig
	if err := json.Unmarshal(provider.Config, &cfg); err != nil {
		return nil, fmt.Errorf("parse aws_role config: %w", err)
	}

	identity, err := v.replaySTS(ctx, sts.SignedHeaders, sts.Region)
	if err != nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidSignature"}, nil
	}

	claims := map[string]interface{}{
		"arn":         identity.Arn,
		"accountId":   identity.Account,
		"username":    identity.UserID,
		"roleArn":     identity.Arn,
		"assumedRole": identity.Arn,
	}

	passed, failure := EvaluateMatchRules(cfg.MatchRules, claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true}, nil
}

type stsIdentity struct {
	Arn     string
	Account string
	UserID  string
}

type getCallerIdentityResponse struct {
	XMLName xml.Name                `xml:"GetCallerIdentityResponse"`
	Result  getCallerIdentityResult `xml:"GetCallerIdentityResult"`
}

type getCallerIdentityResult struct {
	Arn     string `xml:"Arn"`
	Account string `xml:"Account"`
	UserID  string `xml:"UserId"`
}

func (v *AWSRoleValidator) replaySTS(ctx context.Context, signedHeaders map[string]string, region string) (*stsIdentity, error) {
	url := fmt.Sprintf("https://sts.%s.amazonaws.com/", region)
	body := "Action=GetCallerIdentity&Version=2011-06-15"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, value := range signedHeaders {
		req.Header.Set(key, value)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("STS returned %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stsResp getCallerIdentityResponse
	if err := xml.Unmarshal(respBody, &stsResp); err != nil {
		return nil, fmt.Errorf("parse STS response: %w", err)
	}

	return &stsIdentity{
		Arn:     stsResp.Result.Arn,
		Account: stsResp.Result.Account,
		UserID:  stsResp.Result.UserID,
	}, nil
}
