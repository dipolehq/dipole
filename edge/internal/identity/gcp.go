/*
gcp.go gathers GCP identity token evidence from the Google Compute Engine
metadata server. The token is a signed JWT that Cloud validates using Google's
OIDC JWKS endpoint. Returns nil if the metadata server is unreachable (proxy
is not on GCP).
*/
package identity

import (
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	gcpMetadataBaseURL = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/identity"
	gcpTimeout         = 2 * time.Second
)

type GCPEvidence struct {
	IdentityToken string
}

type gcpGatherer struct {
	client   *http.Client
	audience string
}

func newGCPGatherer(tenantID string) *gcpGatherer {
	return &gcpGatherer{
		client:   &http.Client{Timeout: gcpTimeout},
		audience: tenantID,
	}
}

func (g *gcpGatherer) Gather() (*GCPEvidence, error) {
	url := gcpMetadataBaseURL + "?audience=" + g.audience + "&format=full"

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil
	}
	req.Header.Set("Metadata-Flavor", "Google")

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

	token := strings.TrimSpace(string(body))
	if token == "" {
		return nil, nil
	}

	return &GCPEvidence{IdentityToken: token}, nil
}
