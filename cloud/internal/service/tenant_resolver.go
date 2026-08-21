// tenant_resolver.go adapts TenantRepository to the EdgeTenantResolver interface used by the Edge API handler.
package service

import (
	"context"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type TenantResolverAdapter struct {
	repo repository.TenantRepository
}

func NewTenantResolverAdapter(repo repository.TenantRepository) *TenantResolverAdapter {
	return &TenantResolverAdapter{repo: repo}
}

func (a *TenantResolverAdapter) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Tenant, error) {
	return a.repo.GetBySubdomain(ctx, subdomain)
}
