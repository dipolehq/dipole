/*
oidc_handler.go serves OIDC Discovery and JWKS endpoints unconditionally.

These endpoints MUST be available on the main HTTP server regardless of MCP
configuration, because external systems (AWS STS, Azure Entra, GCP WIF) fetch
our JWKS to verify OIDC tokens issued by the Credential Broker.
*/
package rest

import (
	"net/http"
)

type OIDCIssuer interface {
	DiscoveryDocument() map[string]interface{}
	JWKS() map[string]interface{}
	IssuerURL() string
}

type OIDCHandler struct {
	issuer OIDCIssuer
}

func NewOIDCHandler(issuer OIDCIssuer) *OIDCHandler {
	return &OIDCHandler{issuer: issuer}
}

func (h *OIDCHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /.well-known/openid-configuration", h.HandleDiscovery)
	mux.HandleFunc("GET /.well-known/openid-configuration/jwks", h.HandleJWKS)
}

func (h *OIDCHandler) HandleDiscovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.issuer.DiscoveryDocument())
}

func (h *OIDCHandler) HandleJWKS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.issuer.JWKS())
}
