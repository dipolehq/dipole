package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error)
	GetByAPITokenHash(ctx context.Context, hash string) (*domain.Tenant, error)
	List(ctx context.Context, opts domain.ListOptions) ([]domain.Tenant, int, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ClientWorkloadRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ClientWorkload, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ClientWorkload, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type ServerWorkloadRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ServerWorkload, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ServerWorkload, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, sw *domain.ServerWorkload) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type TrustProviderRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.TrustProvider, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.TrustProvider, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, tp *domain.TrustProvider) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type CredentialProviderRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.CredentialProvider, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.CredentialProvider, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, cp *domain.CredentialProvider) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type AccessConditionRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AccessCondition, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessCondition, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AccessCondition) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type AccessPolicyRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, policy *domain.AccessPolicy) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AccessPolicy, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AccessPolicy, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, policy *domain.AccessPolicy) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	FindByServerTarget(ctx context.Context, tenantID uuid.UUID, host string, port int) ([]domain.AccessPolicyWithRelations, error)
}

type AgentControllerRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, ac *domain.AgentController) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.AgentController, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.AgentController, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, ac *domain.AgentController) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	GetByDeviceCode(ctx context.Context, tenantID uuid.UUID, code string) (*domain.AgentController, error)
}

type EventRepository interface {
	InsertAuthEvent(ctx context.Context, tenantID uuid.UUID, event *domain.AuthEvent) error
	InsertWorkloadEvent(ctx context.Context, tenantID uuid.UUID, event *domain.WorkloadEvent) error
	InsertAuditLog(ctx context.Context, tenantID uuid.UUID, log *domain.AuditLog) error
	QueryAuthEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuthEvent, int, error)
	QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error)
	QueryAuditLogs(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuditLog, int, error)
}

type ResourceSetRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ResourceSet, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	GetRoleIDs(ctx context.Context, tenantID uuid.UUID, resourceSetID uuid.UUID) ([]uuid.UUID, error)
}

type UserRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, user *domain.User) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*domain.User, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.User, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, user *domain.User) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type RoleRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.Role, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.Role, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, role *domain.Role) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
	AssignResourceSets(ctx context.Context, tenantID uuid.UUID, roleID uuid.UUID, resourceSetIDs []uuid.UUID) error
	GetResourceSetIDs(ctx context.Context, tenantID uuid.UUID, roleID uuid.UUID) ([]uuid.UUID, error)
}

type IdentityProviderRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.IdentityProvider, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.IdentityProvider, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, idp *domain.IdentityProvider) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type LogStreamRepository interface {
	Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.LogStream, error)
	List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.LogStream, int, error)
	Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}
