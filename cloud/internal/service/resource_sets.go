/*
resource_sets.go implements Resource Set management for enterprise multi-tenancy.

Resource Sets provide isolation within a tenant — each team/environment gets
their own workloads, policies, and credentials. The default Resource Set ID
is ffffffff-ffff-ffff-ffff-ffffffffffff. Component copying between sets allows
sharing configurations across teams without manual recreation.
*/
package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

var DefaultResourceSetID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

type ResourceSetService struct {
	rsRepo  repository.ResourceSetRepository
	cwRepo  repository.ClientWorkloadRepository
	swRepo  repository.ServerWorkloadRepository
	tpRepo  repository.TrustProviderRepository
	cpRepo  repository.CredentialProviderRepository
	apRepo  repository.AccessPolicyRepository
	limiter *PlanLimiter
}

func NewResourceSetService(
	rs repository.ResourceSetRepository,
	cw repository.ClientWorkloadRepository,
	sw repository.ServerWorkloadRepository,
	tp repository.TrustProviderRepository,
	cp repository.CredentialProviderRepository,
	ap repository.AccessPolicyRepository,
) *ResourceSetService {
	return &ResourceSetService{
		rsRepo: rs, cwRepo: cw, swRepo: sw, tpRepo: tp, cpRepo: cp, apRepo: ap,
	}
}

func (s *ResourceSetService) SetLimiter(l *PlanLimiter) { s.limiter = l }

func (s *ResourceSetService) Create(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, ResourceResourceSets); err != nil {
			return err
		}
	}
	rs.ID = uuid.New()
	rs.TenantID = tenantID
	return s.rsRepo.Create(ctx, tenantID, rs)
}

func (s *ResourceSetService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ResourceSet, int, error) {
	return s.rsRepo.List(ctx, tenantID, opts)
}

func (s *ResourceSetService) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.ResourceSet, error) {
	return s.rsRepo.GetByID(ctx, tenantID, id)
}

func (s *ResourceSetService) Update(ctx context.Context, tenantID uuid.UUID, rs *domain.ResourceSet) error {
	return s.rsRepo.Update(ctx, tenantID, rs)
}

func (s *ResourceSetService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	if id == DefaultResourceSetID {
		return fmt.Errorf("cannot delete the default Resource Set")
	}
	return s.rsRepo.Delete(ctx, tenantID, id)
}

type CopyRequest struct {
	SourceResourceSetID uuid.UUID `json:"sourceResourceSetId"`
	EntityType          string    `json:"entityType"`
	EntityID            uuid.UUID `json:"entityId"`
}

func (s *ResourceSetService) CopyComponent(ctx context.Context, tenantID uuid.UUID, targetRSID uuid.UUID, req *CopyRequest) (uuid.UUID, error) {
	switch req.EntityType {
	case "client_workload":
		return s.copyClientWorkload(ctx, tenantID, targetRSID, req.EntityID)
	case "server_workload":
		return s.copyServerWorkload(ctx, tenantID, targetRSID, req.EntityID)
	case "trust_provider":
		return s.copyTrustProvider(ctx, tenantID, targetRSID, req.EntityID)
	case "credential_provider":
		return s.copyCredentialProvider(ctx, tenantID, targetRSID, req.EntityID)
	case "access_policy":
		return s.copyAccessPolicy(ctx, tenantID, targetRSID, req.EntityID)
	default:
		return uuid.Nil, fmt.Errorf("unsupported entity type for copy: %s", req.EntityType)
	}
}

func (s *ResourceSetService) copyClientWorkload(ctx context.Context, tenantID, targetRS, sourceID uuid.UUID) (uuid.UUID, error) {
	source, err := s.cwRepo.GetByID(ctx, tenantID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	copy := *source
	copy.ID = uuid.New()
	copy.ResourceSetID = &targetRS
	copy.Name = source.Name + " (copy)"
	if err := s.cwRepo.Create(ctx, tenantID, &copy); err != nil {
		return uuid.Nil, err
	}
	return copy.ID, nil
}

func (s *ResourceSetService) copyServerWorkload(ctx context.Context, tenantID, targetRS, sourceID uuid.UUID) (uuid.UUID, error) {
	source, err := s.swRepo.GetByID(ctx, tenantID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	copy := *source
	copy.ID = uuid.New()
	copy.ResourceSetID = &targetRS
	copy.Name = source.Name + " (copy)"
	if err := s.swRepo.Create(ctx, tenantID, &copy); err != nil {
		return uuid.Nil, err
	}
	return copy.ID, nil
}

func (s *ResourceSetService) copyTrustProvider(ctx context.Context, tenantID, targetRS, sourceID uuid.UUID) (uuid.UUID, error) {
	source, err := s.tpRepo.GetByID(ctx, tenantID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	copy := *source
	copy.ID = uuid.New()
	copy.ResourceSetID = &targetRS
	copy.Name = source.Name + " (copy)"
	copy.Config = json.RawMessage(append([]byte{}, source.Config...))
	if err := s.tpRepo.Create(ctx, tenantID, &copy); err != nil {
		return uuid.Nil, err
	}
	return copy.ID, nil
}

func (s *ResourceSetService) copyCredentialProvider(ctx context.Context, tenantID, targetRS, sourceID uuid.UUID) (uuid.UUID, error) {
	source, err := s.cpRepo.GetByID(ctx, tenantID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	copy := *source
	copy.ID = uuid.New()
	copy.ResourceSetID = &targetRS
	copy.Name = source.Name + " (copy)"
	copy.ConfigEncrypted = append([]byte{}, source.ConfigEncrypted...)
	if err := s.cpRepo.Create(ctx, tenantID, &copy); err != nil {
		return uuid.Nil, err
	}
	return copy.ID, nil
}

func (s *ResourceSetService) copyAccessPolicy(ctx context.Context, tenantID, targetRS, sourceID uuid.UUID) (uuid.UUID, error) {
	source, err := s.apRepo.GetByID(ctx, tenantID, sourceID)
	if err != nil {
		return uuid.Nil, err
	}
	copy := *source
	copy.ID = uuid.New()
	copy.ResourceSetID = &targetRS
	copy.Name = source.Name + " (copy)"
	if err := s.apRepo.Create(ctx, tenantID, &copy); err != nil {
		return uuid.Nil, err
	}
	return copy.ID, nil
}

// ResolveResourceSetID returns the effective Resource Set ID for a request.
// If provided is nil, returns the default Resource Set ID.
func ResolveResourceSetID(provided *uuid.UUID) uuid.UUID {
	if provided != nil {
		return *provided
	}
	return DefaultResourceSetID
}
