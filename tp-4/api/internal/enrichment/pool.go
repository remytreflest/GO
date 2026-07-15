package enrichment

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"mira/tp-4/api/internal/core"
	"mira/tp-4/api/internal/embedding"
)

const (
	maxTags             = 5
	summaryMaxLen       = 160
	fallbackSaveTimeout = 2 * time.Second
)

// Pool is a bounded worker pool that processes enrichment Jobs off an
// internal channel, so HTTP handlers can enqueue work without blocking on it.
type Pool struct {
	jobs    chan Job
	store   core.EnrichmentStore
	workers int
	timeout time.Duration
	logger  *slog.Logger

	mu     sync.Mutex
	closed bool
	wg     sync.WaitGroup
}

// NewPool creates a Pool. workers is the number of concurrent goroutines
// consuming jobs; queueSize bounds the internal channel so a burst of writes
// can't grow memory unbounded; timeout bounds each job's SaveEnrichment call.
func NewPool(store core.EnrichmentStore, workers, queueSize int, timeout time.Duration, logger *slog.Logger) *Pool {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pool{
		jobs:    make(chan Job, queueSize),
		store:   store,
		workers: workers,
		timeout: timeout,
		logger:  logger,
	}
}

// Start spawns the worker goroutines. ctx bounds their lifetime for the
// underlying SaveEnrichment calls (each job additionally gets its own
// per-job timeout, see process).
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx)
	}
}

func (p *Pool) worker(ctx context.Context) {
	defer p.wg.Done()
	for job := range p.jobs {
		p.process(ctx, job)
	}
}

// process computes the enrichment result (pure, fast, local functions — no
// I/O) and writes it back to the store under a per-job timeout. The timeout
// exists to bound the one operation that can actually be slow: the store
// write (e.g. a Postgres round trip). On any save error (including
// deadline-exceeded), it makes a best-effort attempt to flip the note's
// status to "failed" instead of leaving it stuck at "pending" forever.
func (p *Pool) process(ctx context.Context, job Job) {
	jobCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	text := job.Title + " " + job.Content
	result := core.EnrichmentResult{
		NoteID:    job.NoteID,
		Tags:      embedding.ExtractTags(text, maxTags),
		Summary:   embedding.Summarize(job.Content, summaryMaxLen),
		Score:     embedding.Score(job.Title, job.Content),
		Embedding: embedding.Embed(text),
		Status:    core.EnrichmentDone,
	}

	if err := p.store.SaveEnrichment(jobCtx, result); err != nil {
		p.logger.Warn("enrichment_save_failed", "note_id", job.NoteID, "error", err)

		failCtx, failCancel := context.WithTimeout(context.Background(), fallbackSaveTimeout)
		defer failCancel()
		if ferr := p.store.SaveEnrichment(failCtx, core.EnrichmentResult{NoteID: job.NoteID, Status: core.EnrichmentFailed}); ferr != nil {
			p.logger.Warn("enrichment_mark_failed_failed", "note_id", job.NoteID, "error", ferr)
		}
	}
}

// Submit enqueues a job without blocking the caller: if the internal queue
// is full, or the pool has been shut down, it logs a warning and returns
// false rather than blocking the HTTP handler that called it.
func (p *Pool) Submit(job Job) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return false
	}
	select {
	case p.jobs <- job:
		return true
	default:
		p.logger.Warn("enrichment_queue_full", "note_id", job.NoteID)
		return false
	}
}

// Shutdown stops accepting new jobs and waits for in-flight/queued jobs to
// drain, bounded by ctx. Closing under the same mutex Submit locks avoids a
// send-on-closed-channel panic if Submit races with Shutdown.
func (p *Pool) Shutdown(ctx context.Context) error {
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		close(p.jobs)
	}
	p.mu.Unlock()

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
