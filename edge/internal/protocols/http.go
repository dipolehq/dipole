/*
http.go encapsulates the HTTP request/response forwarding cycle.

HTTPHandler reads an HTTP request from the client, optionally injects
credentials, writes the request to the server, reads the response, and
relays it back. Supports keep-alive by looping until Connection: close.
*/
package protocols

import (
	"bufio"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/edge/internal/injection"
	"github.com/dipolehq/dipole/edge/internal/metrics"
)

type HTTPHandler struct {
	Logger *slog.Logger
}

func (h *HTTPHandler) Handle(clientReader *bufio.Reader, clientWriter io.Writer, serverConn io.ReadWriter, targetHost string, cred *injection.Credential, config *injection.InjectionConfig, connID uuid.UUID) {
	serverReader := bufio.NewReader(serverConn)

	for {
		req, err := http.ReadRequest(clientReader)
		if err != nil {
			if err != io.EOF {
				h.Logger.Debug("read request failed", "host", targetHost, "error", err, "connection_id", connID)
			}
			return
		}

		req.URL.Host = targetHost
		req.URL.Scheme = "https"

		if cred != nil && config != nil {
			if err := injection.InjectCredential(req, cred, config); err != nil {
				h.Logger.Warn("credential injection failed", "method", config.Method, "error", err, "connection_id", connID)
			} else {
				h.Logger.Info("credential injected", "method", config.Method, "host", targetHost, "connection_id", connID)
				metrics.CredentialInjectionsTotal.WithLabelValues("http", config.Method).Inc()
			}
		}

		h.Logger.Info("request", "method", req.Method, "host", targetHost, "path", req.URL.Path, "connection_id", connID)

		if err := req.Write(serverConn); err != nil {
			h.Logger.Debug("write to server failed", "error", err, "connection_id", connID)
			return
		}

		resp, err := http.ReadResponse(serverReader, req)
		if err != nil {
			h.Logger.Debug("read response failed", "error", err, "connection_id", connID)
			return
		}

		if err := resp.Write(clientWriter); err != nil {
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if req.Close || resp.Close || strings.EqualFold(req.Header.Get("Connection"), "close") {
			return
		}
	}
}
