/*
aws_sts.go gathers AWS IAM Role identity evidence by constructing a SigV4-signed
STS GetCallerIdentity request. The signed headers are sent to Cloud which replays
the request against STS to verify the workload's IAM role identity.
Returns nil if IMDS is unreachable or IAM role credentials are unavailable.
*/
package identity

import (
	"encoding/json"
	"strings"
)

type AWSSTSEvidence struct {
	SignedHeaders map[string]string
	Region        string
}

type awsSTSGatherer struct {
	imds *imdsClient
}

func newAWSSTSGatherer(imds *imdsClient) *awsSTSGatherer {
	return &awsSTSGatherer{imds: imds}
}

func (g *awsSTSGatherer) Gather() (*AWSSTSEvidence, error) {
	token, err := g.imds.getSessionToken()
	if err != nil {
		return nil, nil
	}

	regionBytes, err := g.imds.getMetadata(imdsRegionPath, token)
	if err != nil {
		return nil, nil
	}
	region := strings.TrimSpace(string(regionBytes))
	if region == "" {
		return nil, nil
	}

	roleNameBytes, err := g.imds.getMetadata(imdsCredsPath, token)
	if err != nil {
		return nil, nil
	}
	roleName := strings.TrimSpace(string(roleNameBytes))
	if roleName == "" {
		return nil, nil
	}

	credsBytes, err := g.imds.getMetadata(imdsCredsPath+roleName, token)
	if err != nil {
		return nil, nil
	}

	var creds iamCredentials
	if err := json.Unmarshal(credsBytes, &creds); err != nil {
		return nil, nil
	}
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, nil
	}

	headers, err := signSTSGetCallerIdentity(creds.AccessKeyID, creds.SecretAccessKey, creds.Token, region)
	if err != nil {
		return nil, nil
	}

	return &AWSSTSEvidence{
		SignedHeaders: headers,
		Region:        region,
	}, nil
}

type iamCredentials struct {
	AccessKeyID     string `json:"AccessKeyId"`
	SecretAccessKey string `json:"SecretAccessKey"`
	Token           string `json:"Token"`
}
