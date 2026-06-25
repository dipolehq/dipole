/*
aws_imds_client.go provides a shared HTTP client for AWS Instance Metadata
Service (IMDSv2). Both IMDS and STS evidence gatherers use this to avoid
duplicating token acquisition and metadata fetching logic.
*/
package identity

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	imdsBaseURL    = "http://169.254.169.254"
	imdsTokenPath  = "/latest/api/token"
	imdsDocPath    = "/latest/dynamic/instance-identity/document"
	imdsSigPath    = "/latest/dynamic/instance-identity/signature"
	imdsRegionPath = "/latest/meta-data/placement/region"
	imdsCredsPath  = "/latest/meta-data/iam/security-credentials/"
	imdsTimeout    = 2 * time.Second
	imdsTokenTTL   = "21600"
)

type imdsClient struct {
	httpClient *http.Client
	baseURL    string
}

func newIMDSClient() *imdsClient {
	return &imdsClient{
		httpClient: &http.Client{Timeout: imdsTimeout},
		baseURL:    imdsBaseURL,
	}
}

func (c *imdsClient) getSessionToken() (string, error) {
	req, err := http.NewRequest(http.MethodPut, c.baseURL+imdsTokenPath, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", imdsTokenTTL)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("IMDS token request returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *imdsClient) getMetadata(path, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-aws-ec2-metadata-token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDS %s returned %d", path, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
