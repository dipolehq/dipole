package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
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
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "SignOnPolicies", "sign_on_policy", uuid.Nil, "sign-on-policy", clientIP(r), userAgent(r))
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
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "GlobalPolicyCompliance", "global_policy_compliance", uuid.Nil, "global-policy-compliance", clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, map[string]any{
		"apTrustProvider":            req.APTrustProvider,
		"apAccessCondition":          req.APAccessCondition,
		"acTrustProvider":            req.ACTrustProvider,
		"acTlsHostname":              req.ACTLSHostname,
		"accessTokenLifetimeSeconds": req.AccessTokenLifetimeSeconds,
	})
}

// complianceAttributeStatus reports whether a policy actually satisfies an
// enforced attribute (Trust Provider or Access Condition), given the
// tenant's configured enforcement level for that attribute:
//   - "optional": no enforcement — always reported as "optional" regardless
//     of whether the policy has the attribute, matching what the Global
//     Policy Compliance settings page promises ("no enforcement").
//   - "required"/"recommended": if the policy has the attribute, it is
//     compliant; if not, the configured level itself is returned so the
//     frontend can render "Missing Required"/"Missing Recommended".
func complianceAttributeStatus(level string, present bool) string {
	if level == "" {
		level = "optional"
	}
	if level == "optional" {
		return "optional"
	}
	if present {
		return "compliant"
	}
	return level
}

func (h *APIHandlers) GetComplianceReport(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	p, err := h.globalPolicyCompliance.Get(r.Context(), tenantID)
	if err != nil || p == nil {
		respondJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}

	policies, err := h.complianceChecker.ListActiveWithRelations(r.Context(), tenantID)
	if err != nil {
		slog.Error("compliance report: list policies failed", "error", err)
		respondJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}

	resourceSetNames, err := h.resourceSetNames(r.Context(), tenantID)
	if err != nil {
		// Non-fatal: the report is still useful without resource set
		// names — fall back to "Default" for every row rather than
		// failing the whole report over a display-only lookup.
		slog.Error("compliance report: list resource sets failed", "error", err)
		resourceSetNames = map[uuid.UUID]string{}
	}

	type complianceRow struct {
		ID                 uuid.UUID `json:"id"`
		PolicyName         string    `json:"policyName"`
		ClientWorkload     string    `json:"clientWorkload"`
		TrustProvider      string    `json:"trustProvider"`
		AccessCondition    string    `json:"accessCondition"`
		CredentialProvider string    `json:"credentialProvider"`
		ServerWorkload     string    `json:"serverWorkload"`
		ResourceSet        string    `json:"resourceSet"`
	}

	items := make([]complianceRow, 0, len(policies))
	for _, pol := range policies {
		tp := complianceAttributeStatus(p.APTrustProvider, len(pol.TrustProviders) > 0)
		ac := complianceAttributeStatus(p.APAccessCondition, len(pol.AccessConditions) > 0)

		cpName := ""
		if len(pol.CredentialProviders) > 0 {
			cpName = pol.CredentialProviders[0].Name
		}

		rsID := scope.ResolveResourceSetID(pol.Policy.ResourceSetID)
		rsName, ok := resourceSetNames[rsID]
		if !ok {
			rsName = "Default"
		}

		items = append(items, complianceRow{
			ID:                 pol.Policy.ID,
			PolicyName:         pol.Policy.Name,
			ClientWorkload:     pol.ClientWorkload.Name,
			TrustProvider:      tp,
			AccessCondition:    ac,
			CredentialProvider: cpName,
			ServerWorkload:     pol.ServerWorkload.Name,
			ResourceSet:        rsName,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"items": items, "total": len(items)})
}

// resourceSetNames returns every resource set name for the tenant, keyed by
// ID, so GetComplianceReport can render a human-readable name instead of a
// raw UUID. A tenant's resource set count is small (teams/environments, not
// a per-entity resource), so one unpaginated list is a single query and
// safe to hold in memory.
func (h *APIHandlers) resourceSetNames(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]string, error) {
	sets, _, err := h.resourceSets.List(ctx, tenantID, domain.ListOptions{Page: 1, PerPage: 1000})
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(sets))
	for _, rs := range sets {
		names[rs.ID] = rs.Name
	}
	return names, nil
}
