/*
Routes (RegisterAPIRoutes):

	GET    /api/v1/health
	GET/POST /api/v1/agent-controllers, GET/PUT/DELETE /api/v1/agent-controllers/{id}
	GET    /api/v1/agent-proxies
	POST   /api/v1/agent-controllers/{id}/device-code
	GET/POST /api/v1/users, GET/PUT/DELETE /api/v1/users/{id}
	GET/POST /api/v1/roles, GET/PUT/DELETE /api/v1/roles/{id}
	GET/POST /api/v1/resource-sets, GET/PUT/DELETE /api/v1/resource-sets/{id}
	GET/POST /api/v1/identity-providers, GET/PUT/DELETE /api/v1/identity-providers/{id}
	GET/PUT  /api/v1/sign-on-policy
	GET/PUT  /api/v1/global-policy-compliance
	GET      /api/v1/global-policy-compliance/report
	GET/PUT  /api/v1/profile
	POST     /api/v1/profile/token
	GET      /api/v1/profile/tokens
	DELETE   /api/v1/profile/tokens/{id}
	POST     /api/v1/profile/password
	GET/POST /api/v1/log-streams, GET/PUT/DELETE /api/v1/log-streams/{id}
*/
package rest

import (
	"context"
	"net/http"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/events"
	"github.com/dipolehq/dipole/cloud/internal/repository"
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

type ControllerCreator interface {
	CreateController(ctx context.Context, tenantID uuid.UUID, name string) (*domain.AgentController, error)
	RegenerateDeviceCode(ctx context.Context, tenantID, controllerID uuid.UUID) (string, error)
}

type HealthPinger interface {
	Ping(ctx context.Context) error
}

type userService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.User, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	Create(ctx context.Context, tenantID uuid.UUID, u *domain.User) error
	Update(ctx context.Context, tenantID uuid.UUID, u *domain.User) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type roleService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.Role, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.Role, error)
	Create(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error
	Update(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type resourceSetService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ResourceSet, error)
	GetResourceCounts(ctx context.Context, tenantID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]domain.ResourceSetCounts, error)
	Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error
	Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type identityProviderService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.IdentityProvider, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.IdentityProvider, error)
	Create(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error
	Update(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type logStreamService interface {
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.LogStream, int, error)
	Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.LogStream, error)
	Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error
	Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error
	Delete(ctx context.Context, tenantID, id uuid.UUID) error
}

type APIHandlers struct {
	agentControllers       repository.AgentControllerRepository
	agentProxies           repository.AgentProxyRepository
	apiTokens              repository.APITokenRepository
	trustProviders         repository.TrustProviderRepository
	users                  userService
	roles                  roleService
	resourceSets           resourceSetService
	identityProviders      identityProviderService
	signOnPolicies         signOnPolicyStore
	logStreams             logStreamService
	globalPolicyCompliance globalPolicyComplianceStore
	complianceChecker      complianceChecker
	tenantProfiles         tenantProfileStore
	tenants                repository.TenantRepository
	registrationSvc        ControllerCreator
	audit                  *events.EventBus
	healthPinger           HealthPinger
}

func NewAPIHandlers(
	agc repository.AgentControllerRepository,
	ap repository.AgentProxyRepository,
	at repository.APITokenRepository,
	tpr repository.TrustProviderRepository,
	u userService,
	r roleService,
	rs resourceSetService,
	idp identityProviderService,
	sop signOnPolicyStore,
	ls logStreamService,
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
