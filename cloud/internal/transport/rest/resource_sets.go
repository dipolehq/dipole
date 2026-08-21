package rest

import (
	"log/slog"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/scope"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

// --- Resource Set Handlers ---

func (h *APIHandlers) ListResourceSets(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.resourceSets.List(r.Context(), tenantID, opts)
	if err != nil {
		slog.Error("list resource sets failed", "error", err)
		respondError(w, http.StatusInternalServerError, "internal_error", "an internal error occurred")
		return
	}

	// Fetch resource counts for all listed resource sets in one query.
	ids := make([]uuid.UUID, len(items))
	for i := range items {
		ids[i] = items[i].ID
	}
	counts, err := h.resourceSets.GetResourceCounts(r.Context(), tenantID, ids)
	if err != nil {
		slog.Error("get resource counts failed", "error", err)
		// Non-fatal: return list without counts rather than failing entirely.
		counts = nil
	}

	respondJSON(w, http.StatusOK, NewListResponse(toResourceSetResponsesWithCounts(items, counts), total, opts))
}

func (h *APIHandlers) CreateResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	if h.planLimiter != nil {
		if err := h.planLimiter.Check(r.Context(), tenantID, service.ResourceResourceSets); err != nil {
			respondBadRequest(w, err.Error())
			return
		}
	}
	var req resourceSetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	if req.Name == "" {
		respondBadRequest(w, "name is required")
		return
	}
	rs := req.toDomain()
	if err := h.resourceSets.Create(r.Context(), tenantID, &rs); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.resourceSets.GetByID(r.Context(), tenantID, rs.ID)
	if err != nil || saved == nil {
		emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "ResourceSets", "resource_set", rs.ID, rs.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusCreated, toResourceSetResponse(&rs))
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "ResourceSets", "resource_set", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusCreated, toResourceSetResponse(saved))
}

func (h *APIHandlers) GetResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	rs, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if rs == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	respondJSON(w, http.StatusOK, toResourceSetResponse(rs))
}

func (h *APIHandlers) UpdateResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	var req resourceSetRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	rs := req.toDomain()
	rs.ID = id
	rs.TenantID = tenantID
	if err := h.resourceSets.Update(r.Context(), tenantID, &rs); err != nil {
		respondDomainError(w, err)
		return
	}
	saved, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil || saved == nil {
		emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "ResourceSets", "resource_set", rs.ID, rs.Name, clientIP(r), userAgent(r))
		respondJSON(w, http.StatusOK, toResourceSetResponse(&rs))
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "ResourceSets", "resource_set", saved.ID, saved.Name, clientIP(r), userAgent(r))
	respondJSON(w, http.StatusOK, toResourceSetResponse(saved))
}

func (h *APIHandlers) DeleteResourceSet(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	existing, err := h.resourceSets.GetByID(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if existing == nil {
		respondError(w, http.StatusNotFound, "not_found", "resource set not found")
		return
	}
	// The Default Resource Set is where every resource without an explicit
	// assignment lives (enforced at the database column-default level, not
	// just convention) — deleting it would orphan any such resource, so it
	// can never be deleted, mirroring the equivalent guard other resources
	// with a protected default already have in this codebase.
	if existing.ID == scope.DefaultResourceSetID {
		respondError(w, http.StatusBadRequest, "cannot_delete_default", "the Default Resource Set cannot be deleted")
		return
	}
	if err := h.resourceSets.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "ResourceSets", "resource_set", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}

// --- Resource Set DTOs ---

type resourceSetRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (r resourceSetRequest) toDomain() domain.ResourceSet {
	return domain.ResourceSet{
		Name:        r.Name,
		Description: r.Description,
	}
}

type resourceSetResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	IsDefault           bool      `json:"isDefault"`
	AccessPolicies      int       `json:"accessPolicies"`
	ClientWorkloads     int       `json:"clientWorkloads"`
	TrustProviders      int       `json:"trustProviders"`
	AccessConditions    int       `json:"accessConditions"`
	CredentialProviders int       `json:"credentialProviders"`
	ServerWorkloads     int       `json:"serverWorkloads"`
	CreatedAt           string    `json:"createdAt"`
	UpdatedAt           string    `json:"updatedAt"`
}

func toResourceSetResponse(rs *domain.ResourceSet) resourceSetResponse {
	return resourceSetResponse{
		ID:          rs.ID,
		Name:        rs.Name,
		Description: rs.Description,
		IsDefault:   rs.ID == scope.DefaultResourceSetID,
		CreatedAt:   rs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   rs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toResourceSetResponses(sets []domain.ResourceSet) []resourceSetResponse {
	resp := make([]resourceSetResponse, 0, len(sets))
	for i := range sets {
		resp = append(resp, toResourceSetResponse(&sets[i]))
	}
	return resp
}

func toResourceSetResponsesWithCounts(sets []domain.ResourceSet, counts map[uuid.UUID]domain.ResourceSetCounts) []resourceSetResponse {
	resp := make([]resourceSetResponse, 0, len(sets))
	for i := range sets {
		r := toResourceSetResponse(&sets[i])
		if counts != nil {
			c := counts[sets[i].ID]
			r.AccessPolicies = c.AccessPolicies
			r.ClientWorkloads = c.ClientWorkloads
			r.TrustProviders = c.TrustProviders
			r.AccessConditions = c.AccessConditions
			r.CredentialProviders = c.CredentialProviders
			r.ServerWorkloads = c.ServerWorkloads
		}
		resp = append(resp, r)
	}
	return resp
}
