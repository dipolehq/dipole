package events

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakePipelineRepo is a minimal, concurrency-safe Repository stand-in used
// to verify EventBus.Shutdown's drain behavior and WorkloadPipeline's
// event field defaulting.
type fakePipelineRepo struct {
	mu       sync.Mutex
	received []WorkloadEvent
	blockCh  chan struct{}
}

func (f *fakePipelineRepo) InsertSecurityEvent(_ context.Context, _ *SecurityEvent) error {
	return nil
}

func (f *fakePipelineRepo) InsertWorkloadEventBatch(_ context.Context, batch []WorkloadEvent) error {
	if f.blockCh != nil {
		<-f.blockCh
	}
	f.mu.Lock()
	f.received = append(f.received, batch...)
	f.mu.Unlock()
	return nil
}

func (f *fakePipelineRepo) InsertAuditEvent(_ context.Context, _ *AuditEvent) error {
	return nil
}

func (f *fakePipelineRepo) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.received)
}

func (f *fakePipelineRepo) all() []WorkloadEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]WorkloadEvent, len(f.received))
	copy(out, f.received)
	return out
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestEventBus_Shutdown_DrainsBufferedWorkloadEvents(t *testing.T) {
	repo := &fakePipelineRepo{}
	bus := NewEventBus(repo, testLogger())

	bus.EmitWorkload(WorkloadEvent{EventType: "tcp.open"})
	bus.EmitWorkload(WorkloadEvent{EventType: "tcp.close"})

	bus.Shutdown(2 * time.Second)

	if got := repo.count(); got != 2 {
		t.Fatalf("expected 2 events flushed on shutdown, got %d", got)
	}
}

func TestEventBus_Shutdown_TimesOutIfDrainIsSlow(t *testing.T) {
	repo := &fakePipelineRepo{blockCh: make(chan struct{})}
	bus := NewEventBus(repo, testLogger())

	bus.EmitWorkload(WorkloadEvent{EventType: "tcp.open"})

	start := time.Now()
	bus.Shutdown(100 * time.Millisecond)
	elapsed := time.Since(start)

	if elapsed < 100*time.Millisecond {
		t.Fatalf("Shutdown returned before its timeout elapsed: %v", elapsed)
	}
	if elapsed > 1*time.Second {
		t.Fatalf("Shutdown took much longer than its timeout: %v", elapsed)
	}

	close(repo.blockCh)
}

func TestEventBus_Shutdown_NoBufferedEvents_ReturnsQuickly(t *testing.T) {
	repo := &fakePipelineRepo{}
	bus := NewEventBus(repo, testLogger())

	start := time.Now()
	bus.Shutdown(2 * time.Second)
	elapsed := time.Since(start)

	if elapsed > 1*time.Second {
		t.Fatalf("Shutdown with nothing buffered took too long: %v", elapsed)
	}
}

func TestEventBus_EmitWorkload_AssignsEventIDWhenMissing(t *testing.T) {
	repo := &fakePipelineRepo{}
	bus := NewEventBus(repo, testLogger())

	bus.EmitWorkload(WorkloadEvent{EventType: "tcp.open"})
	bus.Shutdown(2 * time.Second)

	got := repo.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].EventID == uuid.Nil {
		t.Error("expected EventID to be auto-assigned when the caller doesn't set one")
	}
}

func TestEventBus_EmitWorkload_PreservesCallerSuppliedEventID(t *testing.T) {
	repo := &fakePipelineRepo{}
	bus := NewEventBus(repo, testLogger())

	want := uuid.New()
	bus.EmitWorkload(WorkloadEvent{EventType: "tcp.open", EventID: want})
	bus.Shutdown(2 * time.Second)

	got := repo.all()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].EventID != want {
		t.Errorf("EventID = %s, want %s (caller-supplied ID must be preserved, not overwritten)", got[0].EventID, want)
	}
}

func TestEventBus_EmitWorkload_ConcurrentWithShutdown_DoesNotPanic(t *testing.T) {
	// Regression test for a real bug found while auditing this change:
	// WorkloadPipeline.Run's ctx.Done() case used to close(p.ch), but
	// Emit's send is `select { case p.ch <- event: ... default: ... }` —
	// sending on a closed channel always panics in Go regardless of
	// select/default, so a goroutine calling EmitWorkload concurrently
	// with Shutdown (e.g. an in-flight request handler that outlives
	// shutdownHTTP's drain window) would have crashed the process the
	// first time this scenario actually occurred in production. Fixed by
	// draining via non-blocking receive instead of closing the channel.
	repo := &fakePipelineRepo{}
	bus := NewEventBus(repo, testLogger())

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				bus.EmitWorkload(WorkloadEvent{EventType: "tcp.open"})
			}
		}
	}()

	// Give the emitter goroutine a moment to actually start racing against
	// Shutdown rather than potentially finishing before it begins.
	time.Sleep(5 * time.Millisecond)
	bus.Shutdown(2 * time.Second)
	close(stop)
	wg.Wait()
}
