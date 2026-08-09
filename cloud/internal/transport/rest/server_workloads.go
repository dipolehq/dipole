/*
Package rest — server_workloads.go is the HTTP handler for the Server
Workloads resource (typed handler → service → repository with DTOs).
*/
package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type serverWorkloadService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ServerWorkload, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ServerWorkload, error)
	Create(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error
	Update(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type ServerWorkloadHandler struct {
	svc   serverWorkloadService
	audit *AuditEmitter
}

func NewServerWorkloadHandler(svc serverWorkloadService) *ServerWorkloadHandler {
	return &ServerWorkloadHandler{svc: svc}
}

func (h *ServerWorkloadHandler) SetAudit(a *AuditEmitter) { h.audit = a }

func (h *ServerWorkloadHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/server-workloads", h.List)
	mux.HandleFunc("POST /api/v1/server-workloads", h.Create)
	mux.HandleFunc("GET /api/v1/server-workloads/{id}", h.Get)
	mux.HandleFunc("PUT /api/v1/server-workloads/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/server-workloads/{id}", h.Delete)
}

type serverWorkloadRequest struct {
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	IsActive          *bool             `json:"isActive"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	AppProtocol       string            `json:"appProtocol"`
	TransportProtocol string            `json:"transportProtocol"`
	TLS               bool              `json:"tls"`
	TLSVerification   string            `json:"tlsVerification"`
	RequestedPort     int               `json:"requestedPort"`
	RequestedTLS      bool              `json:"requestedTls"`
	AuthMethod        string            `json:"authMethod"`
	AuthScheme        string            `json:"authScheme"`
	AuthConfig        map[string]string `json:"authConfig"`
	HTTPHeaders       map[string]string `json:"httpHeaders"`
	URLPath           string            `json:"urlPath"`
	Tags              []tagDTO          `json:"tags"`
	ResourceSetID     *uuid.UUID        `json:"resourceSetId"`
}

type serverWorkloadResponse struct {
	ID                uuid.UUID         `json:"id"`
	Name              string            `json:"name"`
	Description       string            `json:"description"`
	IsActive          bool              `json:"isActive"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	AppProtocol       string            `json:"appProtocol"`
	TransportProtocol string            `json:"transportProtocol"`
	TLS               bool              `json:"tls"`
	TLSVerification   string            `json:"tlsVerification"`
	RequestedPort     int               `json:"requestedPort"`
	RequestedTLS      bool              `json:"requestedTls"`
	AuthMethod        string            `json:"authMethod"`
	AuthScheme        string            `json:"authScheme"`
	AuthConfig        map[string]string `json:"authConfig"`
	HTTPHeaders       map[string]string `json:"httpHeaders"`
	URLPath           string            `json:"urlPath"`
	Tags              []tagDTO          `json:"tags"`
	ResourceSetID     *uuid.UUID        `json:"resourceSetId,omitempty"`
	PolicyCount       int               `json:"policyCount"`
	CreatedAt         time.Time         `json:"createdAt"`
	UpdatedAt         time.Time         `json:"updatedAt"`
}

func (r serverWorkloadRequest) toDomain() domain.ServerWorkload {
	isActive := true
	if r.IsActive != nil {
		isActive = *r.IsActive
	}

	tags := make([]domain.Tag, 0, len(r.Tags))
	for _, t := range r.Tags {
		tags = append(tags, domain.Tag{Key: t.Key, Value: t.Value})
	}

	authConfig := r.AuthConfig
	if authConfig == nil {
		authConfig = map[string]string{}
	}
	httpHeaders := r.HTTPHeaders
	if httpHeaders == nil {
		httpHeaders = map[string]string{}
	}

	return domain.ServerWorkload{
		Name:              r.Name,
		Description:       r.Description,
		IsActive:          isActive,
		Host:              r.Host,
		Port:              r.Port,
		AppProtocol:       r.AppProtocol,
		TransportProtocol: r.TransportProtocol,
		TLS:               r.TLS,
		TLSVerification:   r.TLSVerification,
		RequestedPort:     r.RequestedPort,
		RequestedTLS:      r.RequestedTLS,
		AuthMethod:        r.AuthMethod,
		AuthScheme:        r.AuthScheme,
		AuthConfig:        authConfig,
		HTTPHeaders:       httpHeaders,
		URLPath:           r.URLPath,
		Tags:              tags,
		ResourceSetID:     r.ResourceSetID,
	}
}

func toServerWorkloadResponse(sw *domain.ServerWorkload) serverWorkloadResponse {
	tags := make([]tagDTO, 0, len(sw.Tags))
	for _, t := range sw.Tags {
		tags = append(tags, tagDTO{Key: t.Key, Value: t.Value})
	}

	return serverWorkloadResponse{
		ID:                sw.ID,
		Name:              sw.Name,
		Description:       sw.Description,
		IsActive:          sw.IsActive,
		Host:              sw.Host,
		Port:              sw.Port,
		AppProtocol:       sw.AppProtocol,
		TransportProtocol: sw.TransportProtocol,
		TLS:               sw.TLS,
		TLSVerification:   sw.TLSVerification,
		RequestedPort:     sw.RequestedPort,
		RequestedTLS:      sw.RequestedTLS,
		AuthMethod:        sw.AuthMethod,
		AuthScheme:        sw.AuthScheme,
		AuthConfig:        sw.AuthConfig,
		HTTPHeaders:       sw.HTTPHeaders,
		URLPath:           sw.URLPath,
		Tags:              tags,
		ResourceSetID:     sw.ResourceSetID,
		PolicyCount:       sw.PolicyCount,
		CreatedAt:         sw.CreatedAt,
		UpdatedAt:         sw.UpdatedAt,
	}
}

func (h *ServerWorkloadHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	opts := parseListOptions(r)

	items, total, err := h.svc.List(r.Context(), tenantID, opts)
	if err != nil {
		respondDomainError(w, err)
		return
	}

	resp := make([]serverWorkloadResponse, 0, len(items))
	for i := range items {
		resp = append(resp, toServerWorkloadResponse(&items[i]))
	}
	respondJSON(w, http.StatusOK, NewListResponse(resp, total, opts))
}

func (h *ServerWorkloadHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())

	var req serverWorkloadRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	sw := req.toDomain()
	if err := h.svc.Create(r.Context(), tenantID, &sw); err != nil {
		respondDomainError(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Created(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "server_workload", sw.ID, sw.Name, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusCreated, toServerWorkloadResponse(&sw))
}

func (h *ServerWorkloadHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	sw, err := h.svc.Get(r.Context(), tenantID, id)
	if err != nil {
		respondDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, toServerWorkloadResponse(sw))
}

func (h *ServerWorkloadHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := TenantIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondBadRequest(w, "invalid id")
		return
	}

	var req serverWorkloadRequest
	if err := decodeJSON(r, &req); err != nil {
		respondBadRequest(w, err.Error())
		return
	}

	sw := req.toDomain()
	sw.ID = id
	if err := h.svc.Update(r.Context(), tenantID, &sw); err != nil {
		respondDomainError(w, err)
		return
	}
	if h.audit != nil {
		h.audit.Updated(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "server_workload", sw.ID, sw.Name, clientIP(r), userAgent(r))
	}
	respondJSON(w, http.StatusOK, toServerWorkloadResponse(&sw))
}

func (h *ServerWorkloadHandler) Delete(w http.ResponseWriter, r *http.Request) {
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
	if h.audit != nil {
		h.audit.Deleted(r.Context(), tenantID, UserEmailFromContext(r.Context()), "Workloads", "server_workload", id, "", clientIP(r), userAgent(r))
	}
	w.WriteHeader(http.StatusNoContent)
}
