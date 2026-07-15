package enrichment

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"mira/tp-4/api/internal/core"
)

type fakeEnrichmentStore struct {
	mu     sync.Mutex
	calls  []core.EnrichmentResult
	delay  time.Duration
	onSave chan struct{}
}

func (f *fakeEnrichmentStore) SaveEnrichment(ctx context.Context, result core.EnrichmentResult) error {
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, result)
	f.mu.Unlock()
	if f.onSave != nil {
		f.onSave <- struct{}{}
	}
	return nil
}

func (f *fakeEnrichmentStore) recorded() []core.EnrichmentResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]core.EnrichmentResult, len(f.calls))
	copy(out, f.calls)
	return out
}

func waitSignal(t *testing.T, ch chan struct{}, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		select {
		case <-ch:
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for save #%d", i+1)
		}
	}
}

func TestPool_ProcessesJob(t *testing.T) {
	store := &fakeEnrichmentStore{onSave: make(chan struct{}, 1)}
	pool := NewPool(store, 2, 8, time.Second, nil)
	pool.Start(context.Background())

	if !pool.Submit(Job{NoteID: "1", Title: "Go", Content: "goroutines and channels and worker pools"}) {
		t.Fatalf("Submit should succeed")
	}
	waitSignal(t, store.onSave, 1)

	results := store.recorded()
	if len(results) != 1 {
		t.Fatalf("expected 1 saved result, got %d", len(results))
	}
	r := results[0]
	if r.NoteID != "1" || r.Status != core.EnrichmentDone {
		t.Fatalf("unexpected result: %+v", r)
	}
	if len(r.Tags) == 0 || r.Summary == "" || len(r.Embedding) == 0 {
		t.Fatalf("expected populated enrichment fields, got %+v", r)
	}

	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: unexpected error: %v", err)
	}
}

func TestPool_SaveTimeoutFallsBackToFailed(t *testing.T) {
	store := &fakeEnrichmentStore{delay: 200 * time.Millisecond, onSave: make(chan struct{}, 2)}
	pool := NewPool(store, 1, 8, 20*time.Millisecond, nil)
	pool.Start(context.Background())

	pool.Submit(Job{NoteID: "1", Title: "Go", Content: "slow store"})
	waitSignal(t, store.onSave, 1)

	results := store.recorded()
	if len(results) != 1 {
		t.Fatalf("expected exactly 1 recorded save (the timed-out first attempt should not record), got %d: %+v", len(results), results)
	}
	if results[0].Status != core.EnrichmentFailed {
		t.Fatalf("expected fallback save to mark status failed, got %+v", results[0])
	}

	_ = pool.Shutdown(context.Background())
}

func TestPool_SubmitNonBlockingWhenQueueFull(t *testing.T) {
	store := &fakeEnrichmentStore{}
	pool := NewPool(store, 1, 1, time.Second, nil)
	// Deliberately not started: nothing drains the channel, so we can
	// deterministically fill its buffer without a race against workers.

	if !pool.Submit(Job{NoteID: "1"}) {
		t.Fatalf("first Submit should succeed (buffer has room)")
	}
	if pool.Submit(Job{NoteID: "2"}) {
		t.Fatalf("second Submit should fail: queue is full and nothing is draining it")
	}
}

func TestPool_ShutdownDrainsQueuedJobs(t *testing.T) {
	store := &fakeEnrichmentStore{onSave: make(chan struct{}, 8)}
	pool := NewPool(store, 3, 8, time.Second, nil)
	pool.Start(context.Background())

	for i := 0; i < 5; i++ {
		if !pool.Submit(Job{NoteID: string(rune('a' + i)), Title: "t", Content: "c"}) {
			t.Fatalf("Submit %d should succeed", i)
		}
	}

	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: unexpected error: %v", err)
	}
	if len(store.recorded()) != 5 {
		t.Fatalf("expected all 5 jobs drained before Shutdown returns, got %d", len(store.recorded()))
	}
}

func TestPool_SubmitAfterShutdownReturnsFalse(t *testing.T) {
	store := &fakeEnrichmentStore{}
	pool := NewPool(store, 1, 1, time.Second, nil)
	pool.Start(context.Background())

	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: unexpected error: %v", err)
	}

	if pool.Submit(Job{NoteID: "late"}) {
		t.Fatalf("Submit after Shutdown must return false")
	}
}

func TestPool_ShutdownRespectsContextDeadline(t *testing.T) {
	store := &fakeEnrichmentStore{delay: time.Second}
	pool := NewPool(store, 1, 1, 5*time.Second, nil)
	pool.Start(context.Background())
	pool.Submit(Job{NoteID: "1"})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := pool.Shutdown(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded when workers don't drain in time, got %v", err)
	}
}
