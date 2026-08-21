// log_streams.go implements business logic and validation for Log Streams (CRUD — the export-cycle job lives in service/reporting.LogStreamService).
package resources

import (
	"context"
	"strings"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
	"github.com/dipolehq/dipole/cloud/internal/service"
)

var validLogStreamEventTypes = map[string]struct{}{
	"access_authorization": {},
	"audit_logs":           {},
	"workload_events":      {},
}

var validLogStreamDestinationTypes = map[string]struct{}{
	"aws_s3":           {},
	"gcs_bucket":       {},
	"crowdstrike_siem": {},
	"splunk_siem":      {},
}

type LogStreamService struct {
	repo    repository.LogStreamRepository
	limiter *service.PlanLimiter
}

func NewLogStreamService(repo repository.LogStreamRepository) *LogStreamService {
	return &LogStreamService{repo: repo}
}

func (s *LogStreamService) SetLimiter(l *service.PlanLimiter) { s.limiter = l }

func (s *LogStreamService) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.LogStream, int, error) {
	return s.repo.List(ctx, tenantID, opts)
}

func (s *LogStreamService) Get(ctx context.Context, tenantID, id uuid.UUID) (*domain.LogStream, error) {
	ls, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if ls == nil {
		return nil, domain.ErrNotFound
	}
	return ls, nil
}

func (s *LogStreamService) Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	if s.limiter != nil {
		if err := s.limiter.Check(ctx, tenantID, service.ResourceLogStreams); err != nil {
			return err
		}
	}
	normalizeLogStream(ls)
	if err := validateLogStream(ls); err != nil {
		return err
	}
	return s.repo.Create(ctx, tenantID, ls)
}

func (s *LogStreamService) Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	normalizeLogStream(ls)
	if err := validateLogStream(ls); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, ls.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Update(ctx, tenantID, ls)
}

func (s *LogStreamService) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return domain.ErrNotFound
	}
	return s.repo.Delete(ctx, tenantID, id)
}

func normalizeLogStream(ls *domain.LogStream) {
	ls.Name = strings.TrimSpace(ls.Name)
	ls.Description = strings.TrimSpace(ls.Description)
	ls.EventType = strings.TrimSpace(ls.EventType)
	ls.DestinationType = strings.TrimSpace(ls.DestinationType)
}

func validateLogStream(ls *domain.LogStream) error {
	if ls.Name == "" {
		return domain.NewValidationError("name", "name is required")
	}
	if ls.EventType == "" {
		return domain.NewValidationError("eventType", "eventType is required")
	}
	if _, ok := validLogStreamEventTypes[ls.EventType]; !ok {
		return domain.NewValidationError("eventType", "eventType must be access_authorization, audit_logs, or workload_events")
	}
	if ls.DestinationType == "" {
		return domain.NewValidationError("destinationType", "destinationType is required")
	}
	if _, ok := validLogStreamDestinationTypes[ls.DestinationType]; !ok {
		return domain.NewValidationError("destinationType", "destinationType must be aws_s3, gcs_bucket, crowdstrike_siem, or splunk_siem")
	}
	return nil
}
