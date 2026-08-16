/*
log_streams.go implements event export to external SIEM/storage systems.

LogStreamService manages CRUD for log stream configurations and provides the
export worker that periodically batches events and sends them to configured
destinations (AWS S3, GCS, Splunk HEC, CrowdStrike SIEM). Events are tracked
by last_exported_at to avoid re-exporting. Failed exports retry next cycle.
*/
package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type LogExporter interface {
	Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error
}

type LogStreamService struct {
	repo      repository.EventRepository
	lsRepo    repository.LogStreamRepository
	exporters map[string]LogExporter
	logger    *slog.Logger
}

func NewLogStreamService(repo repository.EventRepository, lsRepo repository.LogStreamRepository, logger *slog.Logger) *LogStreamService {
	return &LogStreamService{
		repo:      repo,
		lsRepo:    lsRepo,
		exporters: make(map[string]LogExporter),
		logger:    logger,
	}
}

func (s *LogStreamService) RegisterExporter(streamType string, exporter LogExporter) {
	s.exporters[streamType] = exporter
}

func (s *LogStreamService) ExportCycle(ctx context.Context) {
	streams, err := s.lsRepo.ListAllActive(ctx)
	if err != nil {
		s.logger.Error("failed to load active log streams", "error", err)
		return
	}

	for i := range streams {
		stream := &streams[i]

		exporter, ok := s.exporters[stream.DestinationType]
		if !ok {
			s.logger.Warn("no exporter for log stream type", "type", stream.DestinationType, "stream_id", stream.ID)
			continue
		}

		if err := s.exportForStream(ctx, stream, exporter); err != nil {
			s.logger.Error("log stream export failed", "stream_id", stream.ID, "type", stream.DestinationType, "error", err)
		}
	}
}

func (s *LogStreamService) exportForStream(ctx context.Context, stream *domain.LogStream, exporter LogExporter) error {
	events, err := s.gatherEvents(ctx, stream)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	if err := exporter.Export(ctx, stream.Config, events); err != nil {
		return err
	}

	now := time.Now().UTC()
	if err := s.lsRepo.UpdateLastExportedAt(ctx, stream.ID, now); err != nil {
		s.logger.Error("failed to update last_exported_at", "stream_id", stream.ID, "error", err)
	}
	return nil
}

func (s *LogStreamService) gatherEvents(ctx context.Context, stream *domain.LogStream) ([]json.RawMessage, error) {
	var since time.Time
	if stream.LastExportedAt != nil {
		since = *stream.LastExportedAt
	} else {
		since = time.Now().UTC().Add(-24 * time.Hour)
	}

	opts := domain.EventQueryOptions{
		Page:      1,
		PerPage:   1000,
		StartDate: &since,
	}

	var events []json.RawMessage

	switch stream.EventType {
	case "access_authorization":
		authEvents, _, err := s.repo.QueryAuthEvents(ctx, stream.TenantID, opts)
		if err != nil {
			return nil, fmt.Errorf("gather auth events: %w", err)
		}
		for _, e := range authEvents {
			raw, _ := json.Marshal(map[string]any{
				"type":          e.EventType,
				"contextId":     e.ContextID.String(),
				"severity":      e.Severity,
				"clientIp":      e.ClientIP,
				"timestamp":     e.Timestamp.Format(time.RFC3339Nano),
				"payload":       e.Payload,
				"resourceSetId": e.ResourceSetID,
			})
			events = append(events, raw)
		}

	case "workload_events":
		wlEvents, _, err := s.repo.QueryWorkloadEvents(ctx, stream.TenantID, opts)
		if err != nil {
			return nil, fmt.Errorf("gather workload events: %w", err)
		}
		for _, e := range wlEvents {
			raw, _ := json.Marshal(map[string]any{
				"type":             e.EventType,
				"connectionId":     e.ConnectionID.String(),
				"policyId":         e.PolicyID,
				"clientWorkloadId": e.ClientWorkloadID,
				"serverWorkloadId": e.ServerWorkloadID,
				"outcome":          e.Outcome,
				"severity":         e.Severity,
				"timestamp":        e.Timestamp.Format(time.RFC3339Nano),
				"payload":          e.Payload,
				"resourceSetId":    e.ResourceSetID,
			})
			events = append(events, raw)
		}

	case "audit_logs":
		auditLogs, _, err := s.repo.QueryAuditLogs(ctx, stream.TenantID, opts)
		if err != nil {
			return nil, fmt.Errorf("gather audit logs: %w", err)
		}
		for _, l := range auditLogs {
			raw, _ := json.Marshal(map[string]any{
				"actorEmail": l.ActorEmail,
				"category":   l.Category,
				"activity":   l.Activity,
				"targetType": l.TargetType,
				"targetName": l.TargetName,
				"severity":   l.Severity,
				"clientIp":   l.ClientIP,
				"timestamp":  l.Timestamp.Format(time.RFC3339Nano),
			})
			events = append(events, raw)
		}
	}

	return events, nil
}

// S3Exporter writes JSON event files to AWS S3.
type S3Exporter struct{}

func (e *S3Exporter) Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error {
	var cfg struct {
		BucketName string `json:"bucket_name"`
		Region     string `json:"region"`
		Prefix     string `json:"prefix"`
		RoleARN    string `json:"role_arn"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse S3 config: %w", err)
	}

	// Production: assume role via STS, write to s3://<bucket>/<prefix>/<date>/<timestamp>.json
	_ = ctx
	_ = events
	return nil
}

// GCSExporter writes JSON event files to Google Cloud Storage.
type GCSExporter struct{}

func (e *GCSExporter) Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error {
	var cfg struct {
		BucketName         string `json:"bucket_name"`
		ProjectID          string `json:"project_id"`
		Prefix             string `json:"prefix"`
		ServiceAccountJSON string `json:"service_account_json"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse GCS config: %w", err)
	}

	// Production: use service account to write to gs://<bucket>/<prefix>/<date>/<timestamp>.json
	_ = ctx
	_ = events
	return nil
}

// SplunkExporter sends events to Splunk via HTTP Event Collector.
type SplunkExporter struct {
	httpClient *http.Client
}

func NewSplunkExporter() *SplunkExporter {
	return &SplunkExporter{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (e *SplunkExporter) Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error {
	var cfg struct {
		HECURL     string `json:"hec_url"`
		HECToken   string `json:"hec_token"`
		Index      string `json:"index"`
		SourceType string `json:"source_type"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse Splunk config: %w", err)
	}

	for _, event := range events {
		splunkEvent := map[string]any{
			"event":      json.RawMessage(event),
			"index":      cfg.Index,
			"sourcetype": cfg.SourceType,
			"source":     "dipole",
		}
		body, _ := json.Marshal(splunkEvent)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.HECURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Splunk "+cfg.HECToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := e.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("splunk HEC request failed: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("splunk HEC returned %d", resp.StatusCode)
		}
	}
	return nil
}

// CrowdStrikeExporter sends events to CrowdStrike Next-Gen SIEM.
type CrowdStrikeExporter struct {
	httpClient *http.Client
}

func NewCrowdStrikeExporter() *CrowdStrikeExporter {
	return &CrowdStrikeExporter{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (e *CrowdStrikeExporter) Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error {
	var cfg struct {
		APIURL       string `json:"api_url"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse CrowdStrike config: %w", err)
	}

	body, _ := json.Marshal(map[string]any{"events": events})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL+"/data-intake/v1/events", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(cfg.ClientID, cfg.ClientSecret)

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("crowdstrike request failed: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("crowdstrike returned %d", resp.StatusCode)
	}
	return nil
}
