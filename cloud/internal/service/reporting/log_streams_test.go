// log_streams_test.go tests config parsing, batching, object key formatting, and export failure/success bookkeeping for the log stream exporters.
package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/domain"
)

func TestParseS3Config_MissingFields(t *testing.T) {
	cases := []struct {
		name   string
		config string
	}{
		{"missing bucket", `{"region":"us-east-1","access_key_id":"a","secret_access_key":"b"}`},
		{"missing region", `{"bucket_name":"b","access_key_id":"a","secret_access_key":"b"}`},
		{"missing access key", `{"bucket_name":"b","region":"us-east-1","secret_access_key":"b"}`},
		{"missing secret key", `{"bucket_name":"b","region":"us-east-1","access_key_id":"a"}`},
		{"empty", `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseS3Config(json.RawMessage(tc.config)); err == nil {
				t.Errorf("expected error for %s", tc.name)
			}
		})
	}
}

func TestParseS3Config_Valid(t *testing.T) {
	cfg, err := parseS3Config(json.RawMessage(`{"bucket_name":"my-bucket","region":"us-east-1","prefix":"logs","access_key_id":"AKIA123","secret_access_key":"secret"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.BucketName != "my-bucket" || cfg.Region != "us-east-1" || cfg.Prefix != "logs" {
		t.Errorf("unexpected parsed config: %+v", cfg)
	}
}

func TestS3Exporter_InvalidConfig_ReturnsErrorWithoutNetworkCall(t *testing.T) {
	e := NewS3Exporter()
	err := e.Export(context.Background(), uuid.New(), json.RawMessage(`{}`), []json.RawMessage{[]byte(`{"a":1}`)})
	if err == nil {
		t.Fatal("expected error for empty S3 config")
	}
}

func TestParseGCSConfig_MissingFields(t *testing.T) {
	cases := []string{
		`{"service_account_json":"{}"}`,
		`{"bucket_name":"b"}`,
		`{}`,
	}
	for _, c := range cases {
		if _, err := parseGCSConfig(json.RawMessage(c)); err == nil {
			t.Errorf("expected error for config %s", c)
		}
	}
}

func TestGCSExporter_InvalidConfig_ReturnsErrorWithoutNetworkCall(t *testing.T) {
	e := NewGCSExporter()
	err := e.Export(context.Background(), uuid.New(), json.RawMessage(`{}`), []json.RawMessage{[]byte(`{"a":1}`)})
	if err == nil {
		t.Fatal("expected error for empty GCS config")
	}
}

func TestBuildJSONLines_OneLinePerEvent(t *testing.T) {
	events := []json.RawMessage{
		[]byte(`{"a":1}`),
		[]byte(`{"b":2}`),
		[]byte(`{"c":3}`),
	}
	out := string(buildJSONLines(events))
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != `{"a":1}` || lines[1] != `{"b":2}` || lines[2] != `{"c":3}` {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestObjectKey_Format(t *testing.T) {
	tenantID := uuid.New()
	key := objectKey("exports", tenantID)

	if !strings.HasPrefix(key, "exports/"+tenantID.String()+"/") {
		t.Errorf("expected key to start with exports/%s/, got %s", tenantID, key)
	}
	if !strings.HasSuffix(key, ".jsonl") {
		t.Errorf("expected .jsonl suffix, got %s", key)
	}
}

func TestObjectKey_NoPrefix(t *testing.T) {
	tenantID := uuid.New()
	key := objectKey("", tenantID)
	if !strings.HasPrefix(key, tenantID.String()+"/") {
		t.Errorf("expected key to start with tenant id with no leading slash, got %s", key)
	}
}

func TestSplunkExporter_BatchesAllEventsIntoOneRequest(t *testing.T) {
	var requestCount int
	var mu sync.Mutex
	var receivedBody string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		if got := r.Header.Get("Authorization"); got != "Splunk test-token" {
			t.Errorf("unexpected Authorization header: %s", got)
		}

		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exporter := NewSplunkExporter()
	config := json.RawMessage(fmt.Sprintf(`{"hec_url":%q,"hec_token":"test-token","index":"main","source_type":"dipole"}`, srv.URL))
	events := []json.RawMessage{
		[]byte(`{"event":1}`),
		[]byte(`{"event":2}`),
		[]byte(`{"event":3}`),
	}

	if err := exporter.Export(context.Background(), uuid.New(), config, events); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requestCount != 1 {
		t.Errorf("expected exactly 1 HTTP request for 3 events (batched), got %d", requestCount)
	}
	lineCount := strings.Count(receivedBody, "\n")
	if lineCount != 3 {
		t.Errorf("expected 3 newline-delimited events in one request body, got %d lines in: %s", lineCount, receivedBody)
	}
}

func TestSplunkExporter_ServerError_ReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	exporter := NewSplunkExporter()
	config := json.RawMessage(fmt.Sprintf(`{"hec_url":%q,"hec_token":"t"}`, srv.URL))
	err := exporter.Export(context.Background(), uuid.New(), config, []json.RawMessage{[]byte(`{}`)})
	if err == nil {
		t.Fatal("expected error on non-200 HEC response")
	}
}

func TestSplunkExporter_MissingConfig_ReturnsErrorWithoutNetworkCall(t *testing.T) {
	exporter := NewSplunkExporter()
	err := exporter.Export(context.Background(), uuid.New(), json.RawMessage(`{}`), []json.RawMessage{[]byte(`{}`)})
	if err == nil {
		t.Fatal("expected error for missing hec_url/hec_token")
	}
}

func TestSplunkExporter_NoEvents_NoRequestSent(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exporter := NewSplunkExporter()
	config := json.RawMessage(fmt.Sprintf(`{"hec_url":%q,"hec_token":"t"}`, srv.URL))
	if err := exporter.Export(context.Background(), uuid.New(), config, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no HTTP request when there are no events")
	}
}

func TestCrowdStrikeExporter_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exporter := NewCrowdStrikeExporter()
	config := json.RawMessage(fmt.Sprintf(`{"api_url":%q,"client_id":"id","client_secret":"secret"}`, srv.URL))
	if err := exporter.Export(context.Background(), uuid.New(), config, []json.RawMessage{[]byte(`{}`)}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCrowdStrikeExporter_NoEvents_NoRequestSent(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exporter := NewCrowdStrikeExporter()
	config := json.RawMessage(fmt.Sprintf(`{"api_url":%q,"client_id":"id","client_secret":"secret"}`, srv.URL))
	if err := exporter.Export(context.Background(), uuid.New(), config, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no HTTP request when there are no events")
	}
}

func TestS3Exporter_NoEvents_NoOp(t *testing.T) {
	e := NewS3Exporter()
	validConfig := json.RawMessage(`{"bucket_name":"b","region":"us-east-1","access_key_id":"a","secret_access_key":"s"}`)
	if err := e.Export(context.Background(), uuid.New(), validConfig, nil); err != nil {
		t.Fatalf("expected no-op (nil error) for empty event batch, got: %v", err)
	}
}

func TestGCSExporter_NoEvents_NoOp(t *testing.T) {
	e := NewGCSExporter()
	validConfig := json.RawMessage(`{"bucket_name":"b","service_account_json":"{}"}`)
	if err := e.Export(context.Background(), uuid.New(), validConfig, nil); err != nil {
		t.Fatalf("expected no-op (nil error) for empty event batch, got: %v", err)
	}
}

type fakeLogStreamRepo struct {
	streams        []domain.LogStream
	lastExportedAt map[uuid.UUID]time.Time
	lastExportErr  map[uuid.UUID]string
}

func newFakeLogStreamRepo(streams []domain.LogStream) *fakeLogStreamRepo {
	return &fakeLogStreamRepo{
		streams:        streams,
		lastExportedAt: make(map[uuid.UUID]time.Time),
		lastExportErr:  make(map[uuid.UUID]string),
	}
}

func (f *fakeLogStreamRepo) Create(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	return nil
}
func (f *fakeLogStreamRepo) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*domain.LogStream, error) {
	return nil, nil
}
func (f *fakeLogStreamRepo) List(ctx context.Context, tenantID uuid.UUID, opts domain.ListOptions) ([]domain.LogStream, int, error) {
	return nil, 0, nil
}
func (f *fakeLogStreamRepo) Update(ctx context.Context, tenantID uuid.UUID, ls *domain.LogStream) error {
	return nil
}
func (f *fakeLogStreamRepo) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}
func (f *fakeLogStreamRepo) ListAllActive(ctx context.Context) ([]domain.LogStream, error) {
	return f.streams, nil
}
func (f *fakeLogStreamRepo) UpdateLastExportedAt(ctx context.Context, id uuid.UUID, t time.Time) error {
	f.lastExportedAt[id] = t
	delete(f.lastExportErr, id)
	return nil
}
func (f *fakeLogStreamRepo) SetLastExportError(ctx context.Context, id uuid.UUID, errMsg string) error {
	f.lastExportErr[id] = errMsg
	return nil
}

type fakeEventRepo struct {
	auditLogs []domain.AuditLog
}

func (f *fakeEventRepo) InsertAuthEvent(ctx context.Context, tenantID uuid.UUID, event *domain.AuthEvent) error {
	return nil
}
func (f *fakeEventRepo) InsertWorkloadEvent(ctx context.Context, tenantID uuid.UUID, event *domain.WorkloadEvent) error {
	return nil
}
func (f *fakeEventRepo) InsertAuditLog(ctx context.Context, tenantID uuid.UUID, log *domain.AuditLog) error {
	return nil
}
func (f *fakeEventRepo) QueryAuthEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuthEvent, int, error) {
	return nil, 0, nil
}
func (f *fakeEventRepo) QueryWorkloadEvents(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.WorkloadEvent, int, error) {
	return nil, 0, nil
}
func (f *fakeEventRepo) QueryAuditLogs(ctx context.Context, tenantID uuid.UUID, opts domain.EventQueryOptions) ([]domain.AuditLog, int, error) {
	return f.auditLogs, len(f.auditLogs), nil
}
func (f *fakeEventRepo) DashboardEventStats(ctx context.Context, tenantID uuid.UUID, timeRange string) (*domain.DashboardEventData, error) {
	return nil, nil
}

type failingExporter struct{ err error }

func (e *failingExporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	return e.err
}

type deadlineCapturingExporter struct {
	hadDeadline bool
	deadline    time.Time
}

func (e *deadlineCapturingExporter) Export(ctx context.Context, tenantID uuid.UUID, config json.RawMessage, events []json.RawMessage) error {
	e.deadline, e.hadDeadline = ctx.Deadline()
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

func TestExportCycle_RecordsErrorOnExportFailure(t *testing.T) {
	streamID := uuid.New()
	streams := []domain.LogStream{{
		ID:              streamID,
		TenantID:        uuid.New(),
		EventType:       "audit_logs",
		DestinationType: "broken_dest",
		IsActive:        true,
	}}
	lsRepo := newFakeLogStreamRepo(streams)
	eventRepo := &fakeEventRepo{auditLogs: []domain.AuditLog{{ActorEmail: "a@b.com", Timestamp: time.Now()}}}

	svc := NewLogStreamService(eventRepo, lsRepo, testLogger())
	svc.RegisterExporter("broken_dest", &failingExporter{err: fmt.Errorf("boom")})

	svc.ExportCycle(context.Background())

	if got := lsRepo.lastExportErr[streamID]; got != "boom" {
		t.Errorf("expected last_export_error to be recorded as %q, got %q", "boom", got)
	}
	if _, ok := lsRepo.lastExportedAt[streamID]; ok {
		t.Error("expected last_exported_at to NOT be set on export failure")
	}
}

func TestExportCycle_ClearsErrorOnSuccess(t *testing.T) {
	streamID := uuid.New()
	streams := []domain.LogStream{{
		ID:              streamID,
		TenantID:        uuid.New(),
		EventType:       "audit_logs",
		DestinationType: "good_dest",
		IsActive:        true,
	}}
	lsRepo := newFakeLogStreamRepo(streams)
	lsRepo.lastExportErr[streamID] = "previous failure"
	eventRepo := &fakeEventRepo{auditLogs: []domain.AuditLog{{ActorEmail: "a@b.com", Timestamp: time.Now()}}}

	svc := NewLogStreamService(eventRepo, lsRepo, testLogger())
	svc.RegisterExporter("good_dest", &failingExporter{err: nil})

	svc.ExportCycle(context.Background())

	if _, stillSet := lsRepo.lastExportErr[streamID]; stillSet {
		t.Error("expected last_export_error to be cleared after a successful export")
	}
	if _, ok := lsRepo.lastExportedAt[streamID]; !ok {
		t.Error("expected last_exported_at to be set after a successful export")
	}
}

func TestExportForStream_AppliesPerStreamTimeout(t *testing.T) {
	streamID := uuid.New()
	streams := []domain.LogStream{{
		ID:              streamID,
		TenantID:        uuid.New(),
		EventType:       "audit_logs",
		DestinationType: "capture_dest",
		IsActive:        true,
	}}
	lsRepo := newFakeLogStreamRepo(streams)
	eventRepo := &fakeEventRepo{auditLogs: []domain.AuditLog{{ActorEmail: "a@b.com", Timestamp: time.Now()}}}

	svc := NewLogStreamService(eventRepo, lsRepo, testLogger())
	exporter := &deadlineCapturingExporter{}
	svc.RegisterExporter("capture_dest", exporter)

	before := time.Now()
	svc.ExportCycle(context.Background())
	after := time.Now()

	if !exporter.hadDeadline {
		t.Fatal("expected exporter to receive a context with a deadline (per-stream export timeout), got none")
	}
	minExpected := before.Add(exportTimeout - 5*time.Second)
	maxExpected := after.Add(exportTimeout + 5*time.Second)
	if exporter.deadline.Before(minExpected) || exporter.deadline.After(maxExpected) {
		t.Errorf("expected deadline ~%s from call time, got %s (call window %s..%s)", exportTimeout, exporter.deadline, before, after)
	}
}

func TestExportCycle_UnknownDestinationType_RecordsError(t *testing.T) {
	streamID := uuid.New()
	streams := []domain.LogStream{{
		ID:              streamID,
		TenantID:        uuid.New(),
		EventType:       "audit_logs",
		DestinationType: "no_such_exporter",
		IsActive:        true,
	}}
	lsRepo := newFakeLogStreamRepo(streams)
	eventRepo := &fakeEventRepo{}

	svc := NewLogStreamService(eventRepo, lsRepo, testLogger())
	svc.ExportCycle(context.Background())

	if got := lsRepo.lastExportErr[streamID]; got == "" {
		t.Error("expected an error to be recorded for a stream with no registered exporter")
	}
}
