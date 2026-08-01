/*
plan_limits.go enforces per-plan creation limits for tenants.

Each plan (starter, teams, enterprise) has a maximum count for each resource
type. The PlanLimiter checks the current count before any Create operation
and returns ErrPlanLimitReached if the tenant is at capacity.
*/
package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

var ErrPlanLimitReached = fmt.Errorf("plan limit reached")

type ResourceType string

const (
	ResourceClientWorkloads     ResourceType = "client_workloads"
	ResourceServerWorkloads     ResourceType = "server_workloads"
	ResourceAccessPolicies      ResourceType = "access_policies"
	ResourceTrustProviders      ResourceType = "trust_providers"
	ResourceCredentialProviders ResourceType = "credential_providers"
	ResourceAccessConditions    ResourceType = "access_conditions"
	ResourceResourceSets        ResourceType = "resource_sets"
	ResourceUsers               ResourceType = "users"
	ResourceAgentControllers    ResourceType = "agent_controllers"
	ResourceLogStreams          ResourceType = "log_streams"
)

type PlanLimits struct {
	ClientWorkloads     int
	ServerWorkloads     int
	AccessPolicies      int
	TrustProviders      int
	CredentialProviders int
	AccessConditions    int
	ResourceSets        int
	Users               int
	AgentControllers    int
	LogStreams          int
	EventRetentionDays  int
}

var planLimitsMap = map[string]PlanLimits{
	"starter": {
		ClientWorkloads:     5,
		ServerWorkloads:     5,
		AccessPolicies:      5,
		TrustProviders:      5,
		CredentialProviders: 5,
		AccessConditions:    5,
		ResourceSets:        1,
		Users:               3,
		AgentControllers:    2,
		LogStreams:          1,
		EventRetentionDays:  7,
	},
	"teams": {
		ClientWorkloads:     50,
		ServerWorkloads:     50,
		AccessPolicies:      50,
		TrustProviders:      50,
		CredentialProviders: 50,
		AccessConditions:    20,
		ResourceSets:        5,
		Users:               20,
		AgentControllers:    10,
		LogStreams:          5,
		EventRetentionDays:  30,
	},
	"enterprise": {
		ClientWorkloads:     -1,
		ServerWorkloads:     -1,
		AccessPolicies:      -1,
		TrustProviders:      -1,
		CredentialProviders: -1,
		AccessConditions:    -1,
		ResourceSets:        -1,
		Users:               -1,
		AgentControllers:    -1,
		LogStreams:          -1,
		EventRetentionDays:  -1,
	},
}

func GetPlanLimits(plan string) PlanLimits {
	if limits, ok := planLimitsMap[plan]; ok {
		return limits
	}
	return planLimitsMap["starter"]
}

func (pl PlanLimits) GetLimit(resource ResourceType) int {
	switch resource {
	case ResourceClientWorkloads:
		return pl.ClientWorkloads
	case ResourceServerWorkloads:
		return pl.ServerWorkloads
	case ResourceAccessPolicies:
		return pl.AccessPolicies
	case ResourceTrustProviders:
		return pl.TrustProviders
	case ResourceCredentialProviders:
		return pl.CredentialProviders
	case ResourceAccessConditions:
		return pl.AccessConditions
	case ResourceResourceSets:
		return pl.ResourceSets
	case ResourceUsers:
		return pl.Users
	case ResourceAgentControllers:
		return pl.AgentControllers
	case ResourceLogStreams:
		return pl.LogStreams
	default:
		return -1
	}
}

type ResourceCounter interface {
	CountResource(ctx context.Context, tenantID uuid.UUID, resource ResourceType) (int, error)
}

type PlanLimiter struct {
	tenants repository.TenantRepository
	counter ResourceCounter
}

func NewPlanLimiter(tenants repository.TenantRepository, counter ResourceCounter) *PlanLimiter {
	return &PlanLimiter{tenants: tenants, counter: counter}
}

func (l *PlanLimiter) Check(ctx context.Context, tenantID uuid.UUID, resource ResourceType) error {
	tenant, err := l.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("resolve tenant plan: %w", err)
	}
	if tenant == nil {
		return fmt.Errorf("tenant not found")
	}

	limits := GetPlanLimits(tenant.Plan)
	limit := limits.GetLimit(resource)

	if limit == -1 {
		return nil
	}

	current, err := l.counter.CountResource(ctx, tenantID, resource)
	if err != nil {
		return fmt.Errorf("count resources: %w", err)
	}

	if current >= limit {
		return fmt.Errorf("%w: %s plan allows max %d %s (current: %d)", ErrPlanLimitReached, tenant.Plan, limit, resource, current)
	}

	return nil
}

type DBResourceCounter struct {
	clientWorkloads     repository.ClientWorkloadRepository
	serverWorkloads     repository.ServerWorkloadRepository
	accessPolicies      repository.AccessPolicyRepository
	trustProviders      repository.TrustProviderRepository
	credentialProviders repository.CredentialProviderRepository
	accessConditions    repository.AccessConditionRepository
	resourceSets        repository.ResourceSetRepository
	users               repository.UserRepository
	agentControllers    repository.AgentControllerRepository
	logStreams          repository.LogStreamRepository
}

func NewDBResourceCounter(
	cw repository.ClientWorkloadRepository,
	sw repository.ServerWorkloadRepository,
	ap repository.AccessPolicyRepository,
	tp repository.TrustProviderRepository,
	cp repository.CredentialProviderRepository,
	ac repository.AccessConditionRepository,
	rs repository.ResourceSetRepository,
	u repository.UserRepository,
	agc repository.AgentControllerRepository,
	ls repository.LogStreamRepository,
) *DBResourceCounter {
	return &DBResourceCounter{
		clientWorkloads:     cw,
		serverWorkloads:     sw,
		accessPolicies:      ap,
		trustProviders:      tp,
		credentialProviders: cp,
		accessConditions:    ac,
		resourceSets:        rs,
		users:               u,
		agentControllers:    agc,
		logStreams:          ls,
	}
}

func (c *DBResourceCounter) CountResource(ctx context.Context, tenantID uuid.UUID, resource ResourceType) (int, error) {
	opts := domain.ListOptions{Page: 1, PerPage: 1}
	var total int
	var err error

	switch resource {
	case ResourceClientWorkloads:
		_, total, err = c.clientWorkloads.List(ctx, tenantID, opts)
	case ResourceServerWorkloads:
		_, total, err = c.serverWorkloads.List(ctx, tenantID, opts)
	case ResourceAccessPolicies:
		_, total, err = c.accessPolicies.List(ctx, tenantID, opts)
	case ResourceTrustProviders:
		_, total, err = c.trustProviders.List(ctx, tenantID, opts)
	case ResourceCredentialProviders:
		_, total, err = c.credentialProviders.List(ctx, tenantID, opts)
	case ResourceAccessConditions:
		_, total, err = c.accessConditions.List(ctx, tenantID, opts)
	case ResourceResourceSets:
		_, total, err = c.resourceSets.List(ctx, tenantID, opts)
	case ResourceUsers:
		_, total, err = c.users.List(ctx, tenantID, opts)
	case ResourceAgentControllers:
		_, total, err = c.agentControllers.List(ctx, tenantID, opts)
	case ResourceLogStreams:
		_, total, err = c.logStreams.List(ctx, tenantID, opts)
	default:
		return 0, fmt.Errorf("unknown resource type: %s", resource)
	}

	if err != nil {
		return 0, err
	}
	return total, nil
}
