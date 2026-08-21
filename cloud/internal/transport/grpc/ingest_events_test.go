// ingest_events_test.go tests the IngestEvents RPC over a real bufconn gRPC connection, specifically that a mid-stream transport error is logged and distinguished from a clean stream close.
package grpc

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/dipolehq/dipole/cloud/internal/events"
	pb "github.com/dipolehq/dipole/gen/dipole/v1"
)

type fakeIngestRepo struct {
	mu       sync.Mutex
	inserted int
}

func (f *fakeIngestRepo) InsertSecurityEvent(_ context.Context, _ *events.SecurityEvent) error {
	return nil
}

func (f *fakeIngestRepo) InsertWorkloadEventBatch(_ context.Context, batch []events.WorkloadEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inserted += len(batch)
	return nil
}

func (f *fakeIngestRepo) InsertAuditEvent(_ context.Context, _ *events.AuditEvent) error {
	return nil
}

func (f *fakeIngestRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.inserted
}

func TestIngestEvents_CleanClose_NoErrorLogged(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	repo := &fakeIngestRepo{}
	bus := events.NewEventBus(repo, logger)
	t.Cleanup(func() { bus.Shutdown(2 * time.Second) })

	server := &PolicyEvaluationServer{events: bus, logger: logger}

	tenantID := uuid.New()
	proxyID := uuid.New()
	sessionToken := generateTestSessionToken(t)
	proxyRepo := newInMemoryProxyRepo()
	proxyRepo.registerWithToken(tenantID, proxyID, sessionToken)

	client := startBufconnServer(t, server, proxyRepo)

	ctx, cancel := context.WithTimeout(withSessionToken(context.Background(), sessionToken), 5*time.Second)
	defer cancel()

	stream, err := client.IngestEvents(ctx)
	if err != nil {
		t.Fatalf("open ingest stream: %v", err)
	}
	if err := stream.Send(&pb.WorkloadEventBatch{
		TenantId: tenantID.String(),
		Events:   []*pb.WorkloadEvent{{EventType: "tcp.open"}},
	}); err != nil {
		t.Fatalf("send batch: %v", err)
	}

	ack, err := stream.CloseAndRecv()
	if err != nil {
		t.Fatalf("CloseAndRecv: %v", err)
	}
	if ack.Accepted != 1 {
		t.Errorf("Accepted = %d, want 1", ack.Accepted)
	}
	if repo.count() != 1 {
		t.Errorf("repo received %d events, want 1", repo.count())
	}
	if strings.Contains(logBuf.String(), "stream receive failed") {
		t.Errorf("clean stream close must not log a receive-failure error, got: %s", logBuf.String())
	}
}

func TestIngestEvents_MidStreamCancellation_LogsError(t *testing.T) {
	var logBuf bytes.Buffer
	var logMu sync.Mutex
	logger := slog.New(slog.NewTextHandler(syncWriter{&logBuf, &logMu}, &slog.HandlerOptions{Level: slog.LevelWarn}))

	repo := &fakeIngestRepo{}
	bus := events.NewEventBus(repo, logger)
	t.Cleanup(func() { bus.Shutdown(2 * time.Second) })

	server := &PolicyEvaluationServer{events: bus, logger: logger}

	tenantID := uuid.New()
	proxyID := uuid.New()
	sessionToken := generateTestSessionToken(t)
	proxyRepo := newInMemoryProxyRepo()
	proxyRepo.registerWithToken(tenantID, proxyID, sessionToken)

	client := startBufconnServer(t, server, proxyRepo)

	ctx, cancel := context.WithCancel(withSessionToken(context.Background(), sessionToken))

	stream, err := client.IngestEvents(ctx)
	if err != nil {
		t.Fatalf("open ingest stream: %v", err)
	}
	if err := stream.Send(&pb.WorkloadEventBatch{
		TenantId: tenantID.String(),
		Events:   []*pb.WorkloadEvent{{EventType: "tcp.open"}},
	}); err != nil {
		t.Fatalf("send batch: %v", err)
	}

	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		logMu.Lock()
		got := logBuf.String()
		logMu.Unlock()
		if strings.Contains(got, "stream receive failed") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	logMu.Lock()
	defer logMu.Unlock()
	t.Fatalf("expected a 'stream receive failed' log entry after mid-stream cancellation, got: %s", logBuf.String())
}

type syncWriter struct {
	w  io.Writer
	mu *sync.Mutex
}

func (s syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
