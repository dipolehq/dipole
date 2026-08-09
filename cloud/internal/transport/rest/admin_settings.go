package rest

import (
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/google/uuid"
)

// --- Sign-On Policy Handlers ---

func (h *APIHandlers) GetSignOnPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	p, err := h.signOnPolicies.Get(r.Context(), tenantID)
	if err != nil {
		slog.Error("get sign-on policy failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"requireSso": p.RequireSSO,
		"requireMfa": p.RequireMFA,
	})
}

func (h *APIHandlers) UpdateSignOnPolicy(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req struct {
		RequireSSO bool `json:"requireSso"`
		RequireMFA bool `json:"requireMfa"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p := &domain.SignOnPolicy{
		TenantID:   tenantID,
		RequireSSO: req.RequireSSO,
		RequireMFA: req.RequireMFA,
	}
	if err := h.signOnPolicies.Upsert(r.Context(), tenantID, p); err != nil {
		slog.Error("update sign-on policy failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "SignOnPolicies", "sign_on_policy", uuid.Nil, "sign-on-policy", clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"requireSso": req.RequireSSO,
		"requireMfa": req.RequireMFA,
	})
}

// --- Global Policy Compliance Handlers ---

func (h *APIHandlers) GetGlobalPolicyCompliance(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	p, err := h.globalPolicyCompliance.Get(r.Context(), tenantID)
	if err != nil {
		slog.Error("get global policy compliance failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"apTrustProvider":            p.APTrustProvider,
		"apAccessCondition":          p.APAccessCondition,
		"acTrustProvider":            p.ACTrustProvider,
		"acTlsHostname":              p.ACTLSHostname,
		"accessTokenLifetimeSeconds": p.AccessTokenLifetimeSeconds,
	})
}

func (h *APIHandlers) UpdateGlobalPolicyCompliance(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req struct {
		APTrustProvider            string `json:"apTrustProvider"`
		APAccessCondition          string `json:"apAccessCondition"`
		ACTrustProvider            string `json:"acTrustProvider"`
		ACTLSHostname              string `json:"acTlsHostname"`
		AccessTokenLifetimeSeconds int    `json:"accessTokenLifetimeSeconds"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p := &domain.GlobalPolicyCompliance{
		TenantID:                   tenantID,
		APTrustProvider:            req.APTrustProvider,
		APAccessCondition:          req.APAccessCondition,
		ACTrustProvider:            req.ACTrustProvider,
		ACTLSHostname:              req.ACTLSHostname,
		AccessTokenLifetimeSeconds: req.AccessTokenLifetimeSeconds,
	}
	if err := h.globalPolicyCompliance.Upsert(r.Context(), tenantID, p); err != nil {
		slog.Error("update global policy compliance failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}
	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "GlobalPolicyCompliance", "global_policy_compliance", uuid.Nil, "global-policy-compliance", clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"apTrustProvider":            req.APTrustProvider,
		"apAccessCondition":          req.APAccessCondition,
		"acTrustProvider":            req.ACTrustProvider,
		"acTlsHostname":              req.ACTLSHostname,
		"accessTokenLifetimeSeconds": req.AccessTokenLifetimeSeconds,
	})
}
