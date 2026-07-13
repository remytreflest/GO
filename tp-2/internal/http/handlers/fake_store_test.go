package handlers

import (
	"errors"
	"net/http"
	"testing"

	"mira/tp-2/internal/core"
	"mira/tp-2/internal/http/middleware"
)

// errStore is a core.Store stub that always fails with a non-ErrNotFound
// error, used to exercise the 500 branches that MemoryStore never triggers.
type errStore struct{}

var errBoom = errors.New("boom")

func (errStore) Create(*core.Note) error                                 { return errBoom }
func (errStore) Get(string) (*core.Note, error)                          { return nil, errBoom }
func (errStore) List(int, int) ([]*core.Note, int, error)                { return nil, 0, errBoom }
func (errStore) Update(string, core.UpdateNoteInput) (*core.Note, error) { return nil, errBoom }
func (errStore) Delete(string) error                                     { return errBoom }
func (errStore) Search(string) ([]*core.Note, error)                     { return nil, errBoom }

var _ core.Store = errStore{}

func TestHandlers_InternalErrorPaths(t *testing.T) {
	mux := middleware.RequestID(NewRouter(errStore{}))

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
