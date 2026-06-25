/*
azure.go gathers Azure instance identity evidence from the Azure Instance
Metadata Service (IMDS). Reads the attested document containing the signed
instance identity. Returns nil if IMDS is unreachable (proxy is not on Azure).
*/
package identity

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

const (
	azureIMDSURL     = "http://169.254.169.254/metadata/attested/document?api-version=2021-02-01"
	azureIMDSTimeout = 2 * time.Second
)

type AzureEvidence struct {
	Encoding  string
	Signature string
	Nonce     string
}

type azureGatherer struct {
	client *http.Client
	url    string
}

func newAzureGatherer() *azureGatherer {
	return &azureGatherer{
		client: &http.Client{Timeout: azureIMDSTimeout},
		url:    azureIMDSURL,
	}
}

func (g *azureGatherer) Gather() (*AzureEvidence, error) {
	req, err := http.NewRequest(http.MethodGet, g.url, nil)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Metadata", "true")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	var doc struct {
		Encoding  string `json:"encoding"`
		Signature string `json:"signature"`
		Nonce     string `json:"nonce"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, nil
	}

	if doc.Signature == "" {
		return nil, nil
	}

	return &AzureEvidence{
		Encoding:  doc.Encoding,
		Signature: doc.Signature,
		Nonce:     doc.Nonce,
	}, nil
}
