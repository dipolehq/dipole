/*
aws_imds.go gathers AWS instance identity evidence from EC2 Instance Metadata
Service (IMDSv2). Reads the instance identity document (JSON) and its RSA
signature. The document is base64-encoded for transport; the signature from
IMDS is already base64-encoded and used as-is. Returns nil if IMDS is
unreachable (proxy is not on AWS).
*/
package identity

import (
	"encoding/base64"
	"strings"
)

type AWSIMDSEvidence struct {
	IdentityDocument  string
	IdentitySignature string
	Region            string
}

type awsIMDSGatherer struct {
	imds *imdsClient
}

func newAWSIMDSGatherer(imds *imdsClient) *awsIMDSGatherer {
	return &awsIMDSGatherer{imds: imds}
}

func (g *awsIMDSGatherer) Gather() (*AWSIMDSEvidence, error) {
	token, err := g.imds.getSessionToken()
	if err != nil {
		return nil, nil
	}

	doc, err := g.imds.getMetadata(imdsDocPath, token)
	if err != nil {
		return nil, nil
	}

	sig, err := g.imds.getMetadata(imdsSigPath, token)
	if err != nil {
		return nil, nil
	}

	region, _ := g.imds.getMetadata(imdsRegionPath, token)

	// Document is JSON text — must be base64-encoded for transport.
	// Signature from IMDS is already base64-encoded text — strip whitespace and use as-is.
	sigStr := strings.ReplaceAll(strings.TrimSpace(string(sig)), "\n", "")

	return &AWSIMDSEvidence{
		IdentityDocument:  base64.StdEncoding.EncodeToString(doc),
		IdentitySignature: sigStr,
		Region:            strings.TrimSpace(string(region)),
	}, nil
}
