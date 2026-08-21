// outbound_http.go provides a single constructor for building an SSRF-protected http.Transport shared by validators making outbound calls.
package trust

import (
	"net"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/service/trust/ssrf"
)

func newSSRFSafeTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = ssrf.SafeDialContext(&net.Dialer{Timeout: 5 * time.Second})
	return t
}
