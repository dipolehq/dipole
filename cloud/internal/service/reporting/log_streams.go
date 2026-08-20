// log_streams.go implements event export to external SIEM/storage systems (AWS S3, GCS, Splunk HEC, CrowdStrike SIEM) — all customer-owned destinations authenticated with customer-supplied, encrypted-at-rest credentials.
package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"google.golang.org/api/option"

	"github.com/dipolehq/dipole/cloud/internal/domain"
	"github.com/dipolehq/dipole/cloud/internal/repository"
)

type LogExporter interface {
	Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error
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
			errMsg := fmt.Sprintf("no exporter registered for destination type %q", stream.DestinationType)
			s.logger.Warn("no exporter for log stream type", "type", stream.DestinationType, "stream_id", stream.ID)
			if err := s.lsRepo.SetLastExportError(ctx, stream.ID, errMsg); err != nil {
				s.logger.Error("failed to record export error", "stream_id", stream.ID, "error", err)
			}
			continue
		}

		if err := s.exportForStream(ctx, stream, exporter); err != nil {
			s.logger.Error("log stream export failed", "stream_id", stream.ID, "type", stream.DestinationType, "error", err)
			if setErr := s.lsRepo.SetLastExportError(ctx, stream.ID, err.Error()); setErr != nil {
				s.logger.Error("failed to record export error", "stream_id", stream.ID, "error", setErr)
			}
		}
	}
}

const exportTimeout = 30 * time.Second

func (s *LogStreamService) exportForStream(ctx context.Context, stream *domain.LogStream, exporter LogExporter) error {
	events, err := s.gatherEvents(ctx, stream)
	if err != nil {
		return err
	}

	if len(events) == 0 {
		return nil
	}

	exportCtx, cancel := context.WithTimeout(ctx, exportTimeout)
	defer cancel()

	if err := exporter.Export(exportCtx, stream.TenantID, stream.Config, events); err != nil {
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

func buildJSONLines(events []json.RawMessage) []byte {
	var buf bytes.Buffer
	for _, event := range events {
		buf.Write(event)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func objectKey(prefix string, tenantID uuid.UUID) string {
	now := time.Now().UTC()
	key := fmt.Sprintf("%s/%s/%s.jsonl", tenantID.String(), now.Format("2006-01-02"), now.Format("20060102T150405.000000000Z"))
	if prefix != "" {
		key = strings.Trim(prefix, "/") + "/" + key
	}
	return key
}

type S3Exporter struct{}

func NewS3Exporter() *S3Exporter { return &S3Exporter{} }

type s3ExporterConfig struct {
	BucketName      string `json:"bucket_name"`
	Region          string `json:"region"`
	Prefix          string `json:"prefix"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

func parseS3Config(config json.RawMessage) (s3ExporterConfig, error) {
	var cfg s3ExporterConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse S3 config: %w", err)
	}
	if cfg.BucketName == "" {
		return cfg, fmt.Errorf("S3 config missing bucket_name")
	}
	if cfg.Region == "" {
		return cfg, fmt.Errorf("S3 config missing region")
	}
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" {
		return cfg, fmt.Errorf("S3 config missing access_key_id/secret_access_key")
	}
	return cfg, nil
}

func (e *S3Exporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	if len(events) == 0 {
		return nil
	}
	cfg, err := parseS3Config(config)
	if err != nil {
		return err
	}

	client := s3.New(s3.Options{
		Region:      cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
	})

	key := objectKey(cfg.Prefix, tenantID)
	body := buildJSONLines(events)

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.BucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/x-ndjson"),
	})
	if err != nil {
		return fmt.Errorf("s3 PutObject failed: %w", err)
	}
	return nil
}

type GCSExporter struct{}

func NewGCSExporter() *GCSExporter { return &GCSExporter{} }

type gcsExporterConfig struct {
	BucketName         string `json:"bucket_name"`
	Prefix             string `json:"prefix"`
	ServiceAccountJSON string `json:"service_account_json"`
}

func parseGCSConfig(config json.RawMessage) (gcsExporterConfig, error) {
	var cfg gcsExporterConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return cfg, fmt.Errorf("parse GCS config: %w", err)
	}
	if cfg.BucketName == "" {
		return cfg, fmt.Errorf("GCS config missing bucket_name")
	}
	if cfg.ServiceAccountJSON == "" {
		return cfg, fmt.Errorf("GCS config missing service_account_json")
	}
	return cfg, nil
}

func (e *GCSExporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	if len(events) == 0 {
		return nil
	}
	cfg, err := parseGCSConfig(config)
	if err != nil {
		return err
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsJSON([]byte(cfg.ServiceAccountJSON)))
	if err != nil {
		return fmt.Errorf("create GCS client: %w", err)
	}
	defer client.Close()

	key := objectKey(cfg.Prefix, tenantID)
	writer := client.Bucket(cfg.BucketName).Object(key).NewWriter(ctx)
	writer.ContentType = "application/x-ndjson"

	body := buildJSONLines(events)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return fmt.Errorf("gcs write failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("gcs write failed: %w", err)
	}
	return nil
}

type SplunkExporter struct {
	httpClient *http.Client
}

func NewSplunkExporter() *SplunkExporter {
	return &SplunkExporter{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (e *SplunkExporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	if len(events) == 0 {
		return nil
	}

	var cfg struct {
		HECURL     string `json:"hec_url"`
		HECToken   string `json:"hec_token"`
		Index      string `json:"index"`
		SourceType string `json:"source_type"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return fmt.Errorf("parse Splunk config: %w", err)
	}
	if cfg.HECURL == "" || cfg.HECToken == "" {
		return fmt.Errorf("Splunk config missing hec_url/hec_token")
	}

	var buf bytes.Buffer
	for _, event := range events {
		splunkEvent := map[string]any{
			"event":      json.RawMessage(event),
			"index":      cfg.Index,
			"sourcetype": cfg.SourceType,
			"source":     "dipole",
		}
		body, err := json.Marshal(splunkEvent)
		if err != nil {
			return fmt.Errorf("marshal splunk event: %w", err)
		}
		buf.Write(body)
		buf.WriteByte('\n')
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.HECURL, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Splunk "+cfg.HECToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("splunk HEC request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("splunk HEC returned %d", resp.StatusCode)
	}
	return nil
}

type CrowdStrikeExporter struct {
	httpClient *http.Client
}

func NewCrowdStrikeExporter() *CrowdStrikeExporter {
	return &CrowdStrikeExporter{httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (e *CrowdStrikeExporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	if len(events) == 0 {
		return nil
	}

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("crowdstrike returned %d", resp.StatusCode)
	}
	return nil
}
