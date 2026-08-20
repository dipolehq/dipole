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
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

// awsRegionPattern matches AWS's own region-name format (e.g. us-east-1,
// ap-south-1, cn-north-1, us-gov-west-1). region is client-supplied
// evidence used directly in URL construction (fmt.Sprintf(v.stsURLFmt,
// region)) — validating it against the shape AWS itself uses closes off
// unvalidated input flowing into that URL, even though the destination
// host is already constrained to the .amazonaws.com suffix.
var awsRegionPattern = regexp.MustCompile(`^[a-z]{2}(-gov)?-[a-z]+-\d$`)

type AWSRoleValidator struct {
	httpClient *http.Client
	stsURLFmt  string // format string for STS URL, default "https://sts.%s.amazonaws.com/"
}

func NewAWSRoleValidator() *AWSRoleValidator {
	return &AWSRoleValidator{
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: newSSRFSafeTransport(),
		},
		stsURLFmt: "https://sts.%s.amazonaws.com/",
	}
}

func (v *AWSRoleValidator) Validate(ctx context.Context, provider *domain.TrustProvider, evidence *domain.ClientEvidence, _ *JWKSCache) (*domain.TrustProviderResult, error) {
	if evidence.AWS == nil || evidence.AWS.STSGetCallerIdentity == nil {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	sts := evidence.AWS.STSGetCallerIdentity
	if len(sts.SignedHeaders) == 0 || sts.Region == "" {
		return &domain.TrustProviderResult{Attested: false, Reason: "NoDataFound"}, nil
	}

	// region is client-supplied and flows directly into URL construction
	// in replaySTS (fmt.Sprintf(v.stsURLFmt, region)) — reject anything
	// that doesn't look like a real AWS region name before it's used,
	// rather than letting arbitrary characters (path separators, query
	// delimiters, etc.) reach URL construction unexamined.
	if !awsRegionPattern.MatchString(sts.Region) {
		return &domain.TrustProviderResult{Attested: false, Reason: "InvalidRegion"}, nil
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

	passed, failure := EvaluateMatchRules(toMatchRules(provider.MatchRules), claims)
	if !passed {
		return &domain.TrustProviderResult{
			Attested:      false,
			Reason:        "MatchRuleFailed",
			Attribute:     failure.Attribute,
			ExpectedValue: failure.ExpectedValue,
			ActualValue:   failure.ActualValue,
		}, nil
	}

	return &domain.TrustProviderResult{Attested: true, Claims: claims}, nil
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

// allowedSigV4Headers is the exact set of headers a genuine SigV4-signed
// GetCallerIdentity request needs. signedHeaders comes directly from
// client-supplied evidence and is replayed verbatim onto the outbound
// request — without this allowlist, a malicious client could smuggle in
// headers like Host or Content-Length that influence request
// routing/parsing, since nothing previously restricted which header
// names evidence was allowed to inject.
var allowedSigV4Headers = map[string]bool{
	"Authorization":        true,
	"X-Amz-Date":           true,
	"X-Amz-Security-Token": true,
	"X-Amz-Content-Sha256": true,
	"Content-Type":         true,
}

func (v *AWSRoleValidator) replaySTS(ctx context.Context, signedHeaders map[string]string, region string) (*stsIdentity, error) {
	url := fmt.Sprintf(v.stsURLFmt, region)
	body := "Action=GetCallerIdentity&Version=2011-06-15"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for key, value := range signedHeaders {
		if !allowedSigV4Headers[http.CanonicalHeaderKey(key)] {
			continue
		}
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
