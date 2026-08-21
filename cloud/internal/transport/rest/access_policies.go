/*
Package rest — access_policies.go is the HTTP handler for the Access
Policies resource (typed handler → service → repository with DTOs).
*/
package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type accessPolicyService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessPolicy, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.AccessPolicy, error)
	Create(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error
	Update(ctx context.Context, tenantID uuid.UUID, p *domain.AccessPolicy) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type AccessPolicyHandler struct {
	svc   accessPolicyService
	audit *events.EventBus
}

func NewAccessPolicyHandler(svc accessPolicyService) *AccessPolicyHandler {
	return &AccessPolicyHandler{svc: svc}
}

func (h *AccessPolicyHandler) SetAudit(a *events.EventBus) { h.audit = a }

func (h *AccessPolicyHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/access-policies", h.List)
	mux.HandleFunc("POST /api/v1/access-policies", h.Create)
	mux.HandleFunc("GET /api/v1/access-policies/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/access-policies/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/access-policies/{id}", h.Delete)
}

type policyCredentialProviderDTO struct {
	CredentialProviderID string `json:"credentialProviderId"`
	MappingType          string `json:"mappingType"`
	MappingValue         string `json:"mappingValue"`
}

type accessPolicyRequest struct {
	Name                  string                        `json:"name"`
	Description           string                        `json:"description"`
	IsActive              *bool                         `json:"isActive"`
	ClientWorkloadID      string                        `json:"clientWorkloadId"`
	ServerWorkloadID      string                        `json:"serverWorkloadId"`
	TrustProviderIDs      []string                      `json:"trustProviderIds"`
	CredentialProviderIDs []policyCredentialProviderDTO `json:"credentialProviderIds"`
	AccessConditionIDs    []string                      `json:"accessConditionIds"`
	Tags                  []tagDTO                      `json:"tags"`
	ResourceSetID         *uuid.UUID                    `json:"resourceSetId"`
}

type accessPolicyResponse struct {
	ID                    uuid.UUID                     `json:"id"`
	Name                  string                        `json:"name"`
	Description           string                        `json:"description"`
	IsActive              bool                          `json:"isActive"`
	ClientWorkloadID      uuid.UUID                     `json:"clientWorkloadId"`
	ServerWorkloadID      uuid.UUID                     `json:"serverWorkloadId"`
	TrustProviderIDs      []string                      `json:"trustProviderIds"`
	CredentialProviderIDs []policyCredentialProviderDTO `json:"credentialProviderIds"`
	AccessConditionIDs    []string                      `json:"accessConditionIds"`
	Tags                  []tagDTO                      `json:"tags"`
	ResourceSetID         *uuid.UUID                    `json:"resourceSetId,omitempty"`
	CreatedAt             time.Time                     `json:"createdAt"`
	UpdatedAt             time.Time                     `json:"updatedAt"`
}

func (r accessPolicyRequest) toDomain() (domain.AccessPolicy, error) {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	cwID, err := uuid.Parse(r.ClientWorkloadID)
	if err != nil {
		cwID = uuid.Nil
	}
	swID, err := uuid.Parse(r.ServerWorkloadID)
	if err != nil {
		swID = uuid.Nil
	}

	var tpIDs []uuid.UUID
	for _, id := range r.TrustProviderIDs {
		if parsed, err := uuid.Parse(id); err == nil {
			tpIDs = append(tpIDs, parsed)
		}
	}

	var cpIDs []domain.PolicyCredentialProvider
	for _, cp := range r.CredentialProviderIDs {
		if parsed, err := uuid.Parse(cp.CredentialProviderID); err == nil {
			cpIDs = append(cpIDs, domain.PolicyCredentialProvider{
				CredentialProviderID: parsed,
				MappingType:          cp.MappingType,
				MappingValue:         cp.MappingValue,
			})
		}
	}

	var acIDs []uuid.UUID
	for _, id := range r.AccessConditionIDs {
		if parsed, err := uuid.Parse(id); err == nil {
			acIDs = append(acIDs, parsed)
		}
	}

	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}

	return domain.AccessPolicy{
		Name:                  r.Name,
		Description:           r.Description,
		IsActive:              isActive,
		ClientWorkloadID:      cwID,
		ServerWorkloadID:      swID,
		TrustProviderIDs:      tpIDs,
		CredentialProviderIDs: cpIDs,
		AccessConditionIDs:    acIDs,
		Tags:                  tags,
		ResourceSetID:         r.ResourceSetID,
	}, nil
}

func toAccessPolicyResponse(p *domain.AccessPolicy) accessPolicyResponse {
	tpIDs := make([]string, 0, len(p.TrustProviderIDs))
	for _, id := range p.TrustProviderIDs {
		tpIDs = append(tpIDs, id.String())
	}

	cpIDs := make([]policyCredentialProviderDTO, 0, len(p.CredentialProviderIDs))
	for _, cp := range p.CredentialProviderIDs {
		cpIDs = append(cpIDs, policyCredentialProviderDTO{
			CredentialProviderID: cp.CredentialProviderID.String(),
			MappingType:          cp.MappingType,
			MappingValue:         cp.MappingValue,
		})
	}

	acIDs := make([]string, 0, len(p.AccessConditionIDs))
	for _, id := range p.AccessConditionIDs {
		acIDs = append(acIDs, id.String())
	}

	tags := make([]tagDTO, 0, len(p.Tags))
	for _, t := range p.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}

	return accessPolicyResponse{
		ID:                    p.ID,
		Name:                  p.Name,
		Description:           p.Description,
		IsActive:              p.IsActive,
		ClientWorkloadID:      p.ClientWorkloadID,
		ServerWorkloadID:      p.ServerWorkloadID,
		TrustProviderIDs:      tpIDs,
		CredentialProviderIDs: cpIDs,
		AccessConditionIDs:    acIDs,
		Tags:                  tags,
		ResourceSetID:         p.ResourceSetID,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
}

func (h *AccessPolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)
	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	resp := make([]accessPolicyResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toAccessPolicyResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *AccessPolicyHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	var req accessPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p, _ := req.toDomain()
	if err := h.svc.Create(r.Context(), tenantID, &p); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditCreated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessPolicies", "access_policy", p.ID, p.Name, clientIP(r), userAgent(r))
	// Read-after-write: return the authoritative state from the database.
	saved, err := h.svc.Get(r.Context(), tenantID, p.ID)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, toAccessPolicyResponse(saved))
}

func (h *AccessPolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	p, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	if p == nil {
		respondError(w, http.StatusNotFound, "not_found", "access policy not found")
		return
	}
	respondJSON(w, http.StatusOK, toAccessPolicyResponse(p))
}

func (h *AccessPolicyHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	var req accessPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}
	p, _ := req.toDomain()
	p.ID = id
	if err := h.svc.Update(r.Context(), tenantID, &p); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditUpdated(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessPolicies", "access_policy", p.ID, p.Name, clientIP(r), userAgent(r))
	// Read-after-write: return the authoritative state from the database.
	saved, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toAccessPolicyResponse(saved))
}

func (h *AccessPolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), tenantID, id); err != nil {
		respondDomainError(w, err)
		return
	}
	emitAuditDeleted(h.audit, r.Context(), tenantID, UserEmailFromContext(r.Context()), "AccessPolicies", "access_policy", id, "", clientIP(r), userAgent(r))
	w.WriteHeader(http.StatusNoContent)
}
