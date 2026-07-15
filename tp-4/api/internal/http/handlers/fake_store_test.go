package handlers

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"mira/tp-4/api/internal/core"
	"mira/tp-4/api/internal/enrichment"
	"mira/tp-4/api/internal/http/middleware"
)

// errStore is a core.Store stub that always fails with a non-ErrNotFound
// error, used to exercise the 500 branches that MemoryStore never triggers.
type errStore struct{}

var errBoom = errors.New("boom")

func (errStore) Create(context.Context, *core.Note) error { return errBoom }
func (errStore) Get(context.Context, string) (*core.Note, error) {
	return nil, errBoom
}
func (errStore) List(context.Context, int, int) ([]*core.Note, int, error) {
	return nil, 0, errBoom
}
func (errStore) Update(context.Context, string, core.UpdateNoteInput) (*core.Note, error) {
	return nil, errBoom
}
func (errStore) Delete(context.Context, string) error { return errBoom }
func (errStore) Search(context.Context, string) ([]*core.Note, error) {
	return nil, errBoom
}

var _ core.Store = errStore{}

// fakeEnricher records submitted jobs instead of running a real pool, so
// handler tests can assert Create/Update trigger enrichment without needing
// a store that also implements core.EnrichmentStore.
type fakeEnricher struct {
	mu   sync.Mutex
	jobs []enrichment.Job
}

func (f *fakeEnricher) Submit(job enrichment.Job) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobs = append(f.jobs, job)
	return true
}

func (f *fakeEnricher) submitted() []enrichment.Job {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]enrichment.Job, len(f.jobs))
	copy(out, f.jobs)
	return out
}

// rejectingEnricher simulates a full queue: Submit always returns false, and
// handlers must still respond successfully (fire-and-forget semantics).
type rejectingEnricher struct{}

func (rejectingEnricher) Submit(enrichment.Job) bool { return false }

func TestHandlers_InternalErrorPaths(t *testing.T) {
	mux := middleware.RequestID(NewRouter(errStore{}, rejectingEnricher{}))

	cases := []struct {
		name   string
		method string
		target string
		body   any
	}{
		{"create", http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}},
		{"list", http.MethodGet, "/api/v1/notes", nil},
		{"get", http.MethodGet, "/api/v1/notes/1", nil},
		{"update", http.MethodPatch, "/api/v1/notes/1", map[string]string{"title": "x"}},
		{"delete", http.MethodDelete, "/api/v1/notes/1", nil},
		{"search", http.MethodGet, "/api/v1/search?q=x", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, mux, tc.method, tc.target, tc.body)
			if rec.Code != http.StatusInternalServerError {
				t.Fatalf("expected 500, got %d (body=%s)", rec.Code, rec.Body.String())
			}
			env := decodeEnvelope(t, rec)
			if env.Error == nil || env.Error.Code != "internal_error" {
				t.Fatalf("expected internal_error, got %+v", env.Error)
			}
		})
	}
}
