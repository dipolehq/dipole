/*
outbound_http.go provides a single constructor for building an
SSRF-protected http.Transport, used by every validator in this package
that makes an outbound HTTP call with tenant/customer-supplied config
(a URL, hostname, or region string that ends up in a request).

Each such validator previously built its own bare &http.Client{...}, which
is the exact duplication-risk pattern this package's re-reviews have
repeatedly caught bugs in: a fix applied to one validator's client
doesn't help the others unless every call site is updated by hand. This
gives every future outbound-HTTP validator a single place to opt into the
safe posture, either at construction (see NewSAMLValidator,
NewAWSRoleValidator) or, for jwks_fetcher.go's already-hardened
ssrfSafeTransport, by inspection that the same technique is used.
*/
package trust

import (
	"net"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/trust/ssrf"
)

// newSSRFSafeTransport returns a clone of http.DefaultTransport with only
// DialContext swapped for ssrf.SafeDialContext. Cloning (rather than
// starting from a bare &http.Transport{}) preserves what a zero-value
// http.Client{} gets for free by falling back to http.DefaultTransport:
// proxy support via HTTP_PROXY/HTTPS_PROXY/NO_PROXY, and standard
// connection-pooling defaults (idle connection reuse, TLS handshake
// timeout). Losing those silently would be a deployment-specific outage
// risk, not just a performance regression — this exact mistake was made
// and caught during the Part 1 SSRF fix for jwks_fetcher.go, and the fix
// there is the reason this constructor exists rather than repeating the
// bare-Transport mistake at each new call site.
func newSSRFSafeTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = ssrf.SafeDialContext(&net.Dialer{Timeout: 5 * time.Second})
	return t
}
