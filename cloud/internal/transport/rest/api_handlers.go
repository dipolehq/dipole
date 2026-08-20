/*
Package rest — api_handlers.go holds the legacy CRUD handlers for resources
not yet migrated to the per-resource handler + service pattern.
*/
package rest

import (
	"context"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
	"github.com/google/uuid"
)

type signOnPolicyStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.SignOnPolicy, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.SignOnPolicy) error
}

type globalPolicyComplianceStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.GlobalPolicyCompliance, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.GlobalPolicyCompliance) error
}

type complianceChecker interface {
	ListActiveWithRelations(ctx context.Context, tenantID uuid.UUID) ([]domain.AccessPolicyWithRelations, error)
}

type tenantProfileStore interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*domain.TenantProfile, error)
	Upsert(ctx context.Context, tenantID uuid.UUID, p *domain.TenantProfile) error
}

// ControllerCreator abstracts the registration service's controller creation
// so the transport layer doesn't depend on the concrete type.
type ControllerCreator interface {
	CreateController(ctx context.Context, tenantID uuid.UUID, name string) (*domain.AgentController, error)
	RegenerateDeviceCode(ctx context.Context, tenantID, controllerID uuid.UUID) (string, error)
}

// HealthPinger checks database connectivity for the health endpoint.
type HealthPinger interface {
	Ping(ctx context.Context) error
}

type APIHandlers struct {
	agentControllers       repository.AgentControllerRepository
	agentProxies           repository.AgentProxyRepository
	apiTokens              repository.APITokenRepository
	trustProviders         repository.TrustProviderRepository
	users                  repository.UserRepository
	roles                  repository.RoleRepository
	resourceSets           repository.ResourceSetRepository
	identityProviders      repository.IdentityProviderRepository
	signOnPolicies         signOnPolicyStore
	logStreams             repository.LogStreamRepository
	globalPolicyCompliance globalPolicyComplianceStore
	complianceChecker      complianceChecker
	tenantProfiles         tenantProfileStore
	tenants                repository.TenantRepository
	registrationSvc        ControllerCreator
	planLimiter            *service.PlanLimiter
	audit                  *events.EventBus
	healthPinger           HealthPinger
}

func NewAPIHandlers(
	agc repository.AgentControllerRepository,
	ap repository.AgentProxyRepository,
	at repository.APITokenRepository,
	tpr repository.TrustProviderRepository,
	u repository.UserRepository,
	r repository.RoleRepository,
	rs repository.ResourceSetRepository,
	idp repository.IdentityProviderRepository,
	sop signOnPolicyStore,
	ls repository.LogStreamRepository,
	gpc globalPolicyComplianceStore,
	cc complianceChecker,
	tp tenantProfileStore,
	regSvc ControllerCreator,
	tenants repository.TenantRepository,
) *APIHandlers {
	return &APIHandlers{
		agentControllers:       agc,
		agentProxies:           ap,
		apiTokens:              at,
		trustProviders:         tpr,
		users:                  u,
		roles:                  r,
		resourceSets:           rs,
		identityProviders:      idp,
		signOnPolicies:         sop,
		logStreams:             ls,
		globalPolicyCompliance: gpc,
		complianceChecker:      cc,
		tenantProfiles:         tp,
		tenants:                tenants,
		registrationSvc:        regSvc,
	}
}

func (h *APIHandlers) SetPlanLimiter(l *service.PlanLimiter) { h.planLimiter = l }

func (h *APIHandlers) SetAudit(a *events.EventBus) { h.audit = a }

func (h *APIHandlers) SetHealthPinger(p HealthPinger) { h.healthPinger = p }

func (h *APIHandlers) RegisterAPIRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/health", h.Health)

	mux.HandleFunc("GET /api/v1/agent-controllers", h.ListAgentControllers)
	mux.HandleFunc("POST /api/v1/agent-controllers", h.CreateAgentController)
	mux.HandleFunc("GET /api/v1/agent-controllers/{id}", h.GetAgentController)
	mux.HandleFunc("PUT /api/v1/agent-controllers/{id}", h.UpdateAgentController)
	mux.HandleFunc("DELETE /api/v1/agent-controllers/{id}", h.DeleteAgentController)
	mux.HandleFunc("GET /api/v1/agent-proxies", h.ListAgentProxies)
	mux.HandleFunc("POST /api/v1/agent-controllers/{id}/device-code", h.RegenerateDeviceCode)

	mux.HandleFunc("GET /api/v1/users", h.ListUsers)
	mux.HandleFunc("POST /api/v1/users", h.CreateUser)
	mux.HandleFunc("GET /api/v1/users/{id}", h.GetUser)
	mux.HandleFunc("PUT /api/v1/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/v1/users/{id}", h.DeleteUser)

	mux.HandleFunc("GET /api/v1/roles", h.ListRoles)
	mux.HandleFunc("POST /api/v1/roles", h.CreateRole)
	mux.HandleFunc("GET /api/v1/roles/{id}", h.GetRole)
	mux.HandleFunc("PUT /api/v1/roles/{id}", h.UpdateRole)
	mux.HandleFunc("DELETE /api/v1/roles/{id}", h.DeleteRole)

	mux.HandleFunc("GET /api/v1/resource-sets", h.ListResourceSets)
	mux.HandleFunc("POST /api/v1/resource-sets", h.CreateResourceSet)
	mux.HandleFunc("GET /api/v1/resource-sets/{id}", h.GetResourceSet)
	mux.HandleFunc("PUT /api/v1/resource-sets/{id}", h.UpdateResourceSet)
	mux.HandleFunc("DELETE /api/v1/resource-sets/{id}", h.DeleteResourceSet)

	mux.HandleFunc("GET /api/v1/identity-providers", h.ListIdentityProviders)
	mux.HandleFunc("POST /api/v1/identity-providers", h.CreateIdentityProvider)
	mux.HandleFunc("GET /api/v1/identity-providers/{id}", h.GetIdentityProvider)
	mux.HandleFunc("PUT /api/v1/identity-providers/{id}", h.UpdateIdentityProvider)
	mux.HandleFunc("DELETE /api/v1/identity-providers/{id}", h.DeleteIdentityProvider)

	mux.HandleFunc("GET /api/v1/sign-on-policy", h.GetSignOnPolicy)
	mux.HandleFunc("PUT /api/v1/sign-on-policy", h.UpdateSignOnPolicy)

	mux.HandleFunc("GET /api/v1/global-policy-compliance", h.GetGlobalPolicyCompliance)
	mux.HandleFunc("PUT /api/v1/global-policy-compliance", h.UpdateGlobalPolicyCompliance)
	mux.HandleFunc("GET /api/v1/global-policy-compliance/report", h.GetComplianceReport)

	mux.HandleFunc("GET /api/v1/profile", h.GetProfile)
	mux.HandleFunc("PUT /api/v1/profile", h.UpdateProfile)
	mux.HandleFunc("POST /api/v1/profile/token", h.GenerateAccessToken)
	mux.HandleFunc("GET /api/v1/profile/tokens", h.ListAccessTokens)
	mux.HandleFunc("DELETE /api/v1/profile/tokens/{id}", h.DeleteAccessToken)
	mux.HandleFunc("POST /api/v1/profile/password", h.UpdatePassword)

	mux.HandleFunc("GET /api/v1/log-streams", h.ListLogStreams)
	mux.HandleFunc("POST /api/v1/log-streams", h.CreateLogStream)
	mux.HandleFunc("GET /api/v1/log-streams/{id}", h.GetLogStream)
	mux.HandleFunc("PUT /api/v1/log-streams/{id}", h.UpdateLogStream)
	mux.HandleFunc("DELETE /api/v1/log-streams/{id}", h.DeleteLogStream)
}

func (h *APIHandlers) Health(w http.ResponseWriter, r *http.Request) {
	if h.healthPinger != nil {
		if err := h.healthPinger.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "Unhealthy", "error": "database unreachable"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "Healthy"})
}
