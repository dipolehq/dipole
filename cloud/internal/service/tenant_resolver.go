/*
tenant_resolver.go adapts TenantRepository to the EdgeTenantResolver interface
used by the Edge API handler.

The Edge API transport defines EdgeTenantResolver without a context parameter.
This adapter bridges that gap by providing context.Background() — acceptable
because tenant resolution is a fast database lookup and the Edge API handler
does not propagate request context into this interface (by design).
*/
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

func (a *TenantResolverAdapter) GetBySubdomain(subdomain string) (*domain.Tenant, error) {
	return a.repo.GetBySubdomain(context.Background(), subdomain)
}
