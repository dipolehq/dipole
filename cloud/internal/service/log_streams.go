/*
log_streams.go implements event export to external SIEM/storage systems.

LogStreamService manages CRUD for log stream configurations and provides the
export worker that periodically batches events and sends them to configured
destinations (AWS S3, GCS, Splunk HEC, CrowdStrike SIEM). Events are tracked
by last_exported_at to avoid re-exporting. Failed exports retry next cycle.
*/
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type LogStreamConfig struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	Name           string
	Type           string
	Config         json.RawMessage
	EventTypes     []string
	Enabled        bool
	LastExportedAt *time.Time
	CreatedAt      time.Time
}

type LogExporter interface {
	Export(ctx context.Context, config json.RawMessage, events []json.RawMessage) error
}

type LogStreamService struct {
	repo      repository.EventRepository
	exporters map[string]LogExporter
	logger    *slog.Logger
}

func NewLogStreamService(repo repository.EventRepository, logger *slog.Logger) *LogStreamService {
	return &LogStreamService{
		repo:      repo,
		exporters: make(map[string]LogExporter),
		logger:    logger,
	}
}

func (s *LogStreamService) RegisterExporter(streamType string, exporter LogExporter) {
	s.exporters[streamType] = exporter
}

func (s *LogStreamService) ExportCycle(ctx context.Context, streams []LogStreamConfig) {
	for i := range streams {
		stream := &streams[i]
		if !stream.Enabled {
			continue
		}

		exporter, ok := s.exporters[stream.Type]
		if !ok {
			s.logger.Warn("no exporter for log stream type", "type", stream.Type)
			continue
		}

		if err := s.exportForStream(ctx, stream, exporter); err != nil {
			s.logger.Error("log stream export failed", "stream_id", stream.ID, "type", stream.Type, "error", err)
		}
	}
}

func (s *LogStreamService) exportForStream(ctx context.Context, stream *LogStreamConfig, exporter LogExporter) error {
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
	stream.LastExportedAt = &now
	return nil
}

func (s *LogStreamService) gatherEvents(_ context.Context, stream *LogStreamConfig) ([]json.RawMessage, error) {
	// Placeholder: gather events from DB since last_exported_at
	// In production: query auth_events + workload_events + audit_logs tables
	// where timestamp > stream.LastExportedAt, limit 1000
	_ = stream
	return nil, nil
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
