package resources

import (
	"context"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

type fakeTenantRepoForLimiter struct {
	plan string
}

func (f *fakeTenantRepoForLimiter) Create(_ context.Context, _ *domain.Tenant) error { return nil }
func (f *fakeTenantRepoForLimiter) GetByID(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return &domain.Tenant{ID: id, Plan: f.plan}, nil
}
func (f *fakeTenantRepoForLimiter) GetBySubdomain(_ context.Context, _ string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForLimiter) GetByAPITokenHash(_ context.Context, _ string) (*domain.Tenant, error) {
	return nil, nil
}
func (f *fakeTenantRepoForLimiter) List(_ context.Context, _ domain.ListOptions) ([]domain.Tenant, int, error) {
	return nil, 0, nil
}
func (f *fakeTenantRepoForLimiter) Update(_ context.Context, _ *domain.Tenant) error { return nil }
func (f *fakeTenantRepoForLimiter) Delete(_ context.Context, _ uuid.UUID) error      { return nil }

type fakeResourceCounterAtLimit struct{}

func (f *fakeResourceCounterAtLimit) CountResource(_ context.Context, _ uuid.UUID, resource service.ResourceType) (int, error) {
	return service.GetPlanLimits("starter").GetLimit(resource), nil
}

func denyingPlanLimiterForTest() *service.PlanLimiter {
	return service.NewPlanLimiter(&fakeTenantRepoForLimiter{plan: "starter"}, &fakeResourceCounterAtLimit{})
}
