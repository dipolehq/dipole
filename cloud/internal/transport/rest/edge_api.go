/*
Routes:

	POST /edge/v1/auth
	POST /edge/v1/credentials
*/
package rest

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service/identity"
)

type EdgePolicyEvaluator interface {
	Evaluate(ctx context.Context, tenantID uuid.UUID, evidence *domain.ClientEvidence, target *domain.ServerTarget) (*domain.EvaluationResult, error)
}

type EdgeTrustValidator interface {
	ValidateByID(tenantID uuid.UUID, trustProviderID uuid.UUID, evidence *domain.ClientEvidence) (*domain.TrustProviderResult, error)
}

type EdgeTenantResolver interface {
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error)
}

type EdgeAPIHandler struct {
	issuer         *identity.IdentityIssuer
	trustValidator EdgeTrustValidator
	policyEval     EdgePolicyEvaluator
	tenantResolver EdgeTenantResolver
}

func NewEdgeAPIHandler(issuer *identity.IdentityIssuer, trust EdgeTrustValidator, policy EdgePolicyEvaluator, tenants EdgeTenantResolver) *EdgeAPIHandler {
	return &EdgeAPIHandler{
		issuer:         issuer,
		trustValidator: trust,
		policyEval:     policy,
		tenantResolver: tenants,
	}
}

func (h *EdgeAPIHandler) RegisterEdgeRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /edge/v1/auth", h.HandleAuth)
	mux.HandleFunc("POST /edge/v1/credentials", h.HandleCredentials)
}

func (h *EdgeAPIHandler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") != "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{
			"success": false,
			"message": "do not set Authorization header on /edge/v1/auth — identity is in the body",
		})
		return
	}

	var req edgeAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": "invalid request body"})
		return
	}

	parsed, err := parseEdgeSDKClientID(req.ClientID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"success": false, "message": err.Error()})
		return
	}

	tenant, err := h.tenantResolver.GetBySubdomain(r.Context(), parsed.TenantSubdomain)
	if err != nil || tenant == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "Authentication failed"})
		return
	}

	evidence := buildEdgeEvidence(req.Client, parsed.IdentityType)

	trustResult, err := h.trustValidator.ValidateByID(tenant.ID, parsed.TrustProviderID, evidence)
	if err != nil || trustResult == nil || !trustResult.Attested {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"success": false, "message": "Authentication failed"})
		return
	}

	token, err := h.issuer.IssueToken(identity.IssuanceRequest{
		Subject:  req.ClientID,
		Audience: "edge-api",
		Lifetime: 1 * time.Hour,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"success": false, "message": "token generation failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken": token,
		"tokenType":   "Bearer",
		"expiresIn":   3600,
	})
}

func (h *EdgeAPIHandler) HandleCredentials(w http.ResponseWriter, r *http.Request) {
	token := extractAPIToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "missing Authorization header"})
		return
	}

	if err := h.issuer.VerifyTokenSignature(token); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid token"})
		return
	}

	claims := decodeTokenClaims(token)
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid token claims"})
		return
	}

	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "token expired"})
			return
		}
	}

	var req edgeCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
		return
	}

	clientID, _ := claims["sub"].(string)
	parsed, err := parseEdgeSDKClientID(clientID)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid token claims"})
		return
	}

	tenant, err := h.tenantResolver.GetBySubdomain(r.Context(), parsed.TenantSubdomain)
	if err != nil || tenant == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "invalid tenant"})
		return
	}

	evidence := buildEdgeEvidence(req.Client, parsed.IdentityType)
	target := &domain.ServerTarget{Host: req.Server.Host, Port: req.Server.Port}

	result, err := h.policyEval.Evaluate(r.Context(), tenant.ID, evidence, target)
	if err != nil || result == nil || !result.Allowed {
		writeJSON(w, http.StatusOK, map[string]any{"credentialType": "Unknown", "data": map[string]any{}})
		return
	}

	writeJSON(w, http.StatusOK, formatCredentialResponse(result.Credential, req.CredentialType))
}

type edgeAuthRequest struct {
	ClientID string          `json:"clientId"`
	Client   edgeClientBlock `json:"client"`
}

type edgeCredentialRequest struct {
	Client         edgeClientBlock `json:"client"`
	Server         edgeServerBlock `json:"server"`
	CredentialType string          `json:"credentialType"`
}

type edgeClientBlock struct {
	GitHub      *edgeIdentityToken   `json:"github,omitempty"`
	GitLab      *edgeIdentityToken   `json:"gitlab,omitempty"`
	Terraform   *edgeIdentityToken   `json:"terraform,omitempty"`
	OIDC        *edgeIdentityToken   `json:"oidc,omitempty"`
	Certificate *edgeCertificateAttn `json:"certificate,omitempty"`
}

type edgeIdentityToken struct {
	IdentityToken string `json:"identityToken"`
}

type edgeCertificateAttn struct {
	CertificatePEM string `json:"certificatePem"`
	Timestamp      int64  `json:"timestamp"`
	Signature      string `json:"signature"`
}

type edgeServerBlock struct {
	Host              string `json:"host"`
	Port              int    `json:"port"`
	TransportProtocol string `json:"transportProtocol"`
}

type parsedClientID struct {
	Region          string
	TenantSubdomain string
	IdentityType    string
	TrustProviderID uuid.UUID
}

func parseEdgeSDKClientID(clientID string) (*parsedClientID, error) {

	parts := strings.Split(clientID, ":")
	if len(parts) != 6 || parts[0] != "dipole" || parts[3] != "identity" {
		return nil, errInvalidClientID
	}

	tpID, err := uuid.Parse(parts[5])
	if err != nil {
		return nil, errInvalidClientID
	}

	return &parsedClientID{
		Region:          parts[1],
		TenantSubdomain: parts[2],
		IdentityType:    parts[4],
		TrustProviderID: tpID,
	}, nil
}

var errInvalidClientID = &jsonError{"invalid clientId format — expected dipole:<region>:<tenant>:identity:<type>:<uuid>"}

type jsonError struct{ msg string }

func (e *jsonError) Error() string { return e.msg }

func buildEdgeEvidence(client edgeClientBlock, identityType string) *domain.ClientEvidence {
	evidence := &domain.ClientEvidence{}

	switch identityType {
	case "github_idtoken":
		if client.GitHub != nil {
			evidence.GitHub = &domain.IdentityTokenEvidence{IdentityToken: client.GitHub.IdentityToken}
		}
	case "gitlab_idtoken":
		if client.GitLab != nil {
			evidence.GitLab = &domain.IdentityTokenEvidence{IdentityToken: client.GitLab.IdentityToken}
		}
	case "terraform_cloud_idtoken":
		if client.Terraform != nil {
			evidence.Terraform = &domain.IdentityTokenEvidence{IdentityToken: client.Terraform.IdentityToken}
		}
	case "oidc_id_token":
		if client.OIDC != nil {
			evidence.Environment = map[string]string{"OIDC_ID_TOKEN": client.OIDC.IdentityToken}
		}
	case "certificate":
		if client.Certificate != nil {
			sig, err := base64.StdEncoding.DecodeString(client.Certificate.Signature)
			if err == nil {
				evidence.CertificateAttestation = &domain.CertificateAttestation{
					CertificatePEM: client.Certificate.CertificatePEM,
					Timestamp:      client.Certificate.Timestamp,
					Signature:      sig,
				}
			}

		}
	}

	return evidence
}

func formatCredentialResponse(cred *domain.Credential, requestedType string) map[string]any {
	if cred == nil {
		return map[string]any{"credentialType": "Unknown", "data": map[string]any{}}
	}

	switch requestedType {
	case "ApiKey":
		return map[string]any{
			"credentialType": "ApiKey",
			"expiresAt":      time.Unix(cred.ExpiresAt, 0).UTC().Format(time.RFC3339),
			"data":           map[string]any{"apiKey": cred.Value},
		}
	case "UsernamePassword":
		return map[string]any{
			"credentialType": "UsernamePassword",
			"data":           map[string]any{"username": cred.Username, "password": cred.Password},
		}
	case "OAuthToken":
		return map[string]any{
			"credentialType": "OAuthToken",
			"expiresAt":      time.Unix(cred.ExpiresAt, 0).UTC().Format(time.RFC3339),
			"data":           map[string]any{"token": cred.Value},
		}
	case "AwsStsFederation":
		return map[string]any{
			"credentialType": "AwsStsFederation",
			"data": map[string]any{
				"awsAccessKeyId":     cred.AWSAccessKeyID,
				"awsSecretAccessKey": cred.AWSSecretAccessKey,
				"awsSessionToken":    cred.AWSSessionToken,
			},
		}
	case "GoogleWorkloadIdentityFederation":
		return map[string]any{
			"credentialType": "GoogleWorkloadIdentityFederation",
			"data":           map[string]any{"token": cred.Value},
		}
	default:
		return map[string]any{"credentialType": "Unknown", "data": map[string]any{}}
	}
}

func decodeTokenClaims(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	json.Unmarshal(payload, &claims)
	return claims
}
