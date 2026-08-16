/*
Package resources — client_workloads.go implements business logic and
validation for Client Workloads.
*/
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

var validClientIdentifierTypes = map[string]struct{}{
	"dipole_client_id":                {},
	"k8s_namespace":                   {},
	"k8s_pod_name":                    {},
	"k8s_pod_name_prefix":             {},
	"k8s_service_account_name":        {},
	"k8s_service_account_uid":         {},
	"aws_account_id":                  {},
	"aws_ec2_instance_id":             {},
	"aws_ecs_task_family":             {},
	"aws_ecs_service_name":            {},
	"aws_lambda_arn":                  {},
	"aws_region":                      {},
	"azure_subscription_id":           {},
	"azure_vm_id":                     {},
	"gcp_identity_token":              {},
	"github_id_token_repository":      {},
	"github_id_token_subject":         {},
	"gitlab_id_token_namespace_path":  {},
	"gitlab_id_token_project_path":    {},
	"gitlab_id_token_ref_path":        {},
	"gitlab_id_token_subject":         {},
	"terraform_cloud_organization_id": {},
	"terraform_cloud_project_id":      {},
	"terraform_cloud_workspace_id":    {},
	"oidc_id_token":                   {},
	"oidc_id_token_audience":          {},
	"oidc_id_token_issuer":            {},
	"oidc_id_token_subject":           {},
	"process_name":                    {},
	"process_path":                    {},
	"process_command_line":            {},
	"process_user_name":               {},
	"hostname":                        {},
	"source_ip":                       {},
	"redirect_uri":                    {},
}

type ClientWorkloadService struct {
	repo    repository.ClientWorkloadRepository
	limiter *service.PlanLimiter
}

func NewClientWorkloadService(repo repository.ClientWorkloadRepository) *ClientWorkloadService {
	return &ClientWorkloadService{repo: repo}
}

func (s *ClientWorkloadService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *ClientWorkloadService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.ClientWorkload, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *ClientWorkloadService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.ClientWorkload, error) {
	cw, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if cw == nil {
		return nil, domain.ErrNotFound
	}
	return cw, nil
}

func (s *ClientWorkloadService) Create(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceClientWorkloads); err != nil {
			return err
		}
	}
	normalizeClientWorkload(cw)
	if err := validateClientWorkload(cw); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, cw)
}

func (s *ClientWorkloadService) Update(ctx context.Context, tenantID uuid.UUID, cw *domain.ClientWorkload) error {
	normalizeClientWorkload(cw)
	if err := validateClientWorkload(cw); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, cw.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, cw)
}

func (s *ClientWorkloadService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeClientWorkload(cw *domain.ClientWorkload) {
	cw.Name = strings.TrimSpace(cw.Name)
	cw.Description = strings.TrimSpace(cw.Description)
	for i := range cw.Identifiers {
		cw.Identifiers[i].Type = strings.TrimSpace(cw.Identifiers[i].Type)
		cw.Identifiers[i].Value = strings.TrimSpace(cw.Identifiers[i].Value)
	}
}

func validateClientWorkload(cw *domain.ClientWorkload) error {
	if cw.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if len(cw.Identifiers) == 0 {
		return domain.NewValidationError("identifiers", "at least one identifier is required")
	}
	for _, id := range cw.Identifiers {
		if id.Type == "" {
			return domain.NewValidationError("identifiers", "identifier type is required")
		}
		if _, ok := validClientIdentifierTypes[id.Type]; !ok {
			return domain.NewValidationError("identifiers", "unknown identifier type: "+id.Type)
		}
		if id.Value == "" {
			return domain.NewValidationError("identifiers", "identifier value is required for "+id.Type)
		}
	}
	return nil
}
