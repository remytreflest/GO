package miraclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func writeEnvelope(t *testing.T, w http.ResponseWriter, status int, data any, apiErr *apiError) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := envelope{Error: apiErr}
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			t.Fatalf("marshal fixture data: %v", err)
		}
		env.Data = b
	}
	_ = json.NewEncoder(w).Encode(env)
}

func TestCreate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/notes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["title"] != "Go" || body["content"] != "notes" {
			t.Fatalf("unexpected body: %v", body)
		}
		writeEnvelope(t, w, http.StatusCreated, Note{
			ID: "server-id", Title: "Go", Content: "notes", EnrichmentStatus: "pending",
		}, nil)
	}))
	defer srv.Close()

	n, err := New(srv.URL, nil).Create(context.Background(), "Go", "notes", nil)
	if err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}
	if n.ID != "server-id" || n.EnrichmentStatus != "pending" {
		t.Fatalf("unexpected note: %+v", n)
	}
}

func TestCreate_ValidationErrorMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusBadRequest, nil, &apiError{Code: "validation_error", Message: "title is required"})
	}))
	defer srv.Close()

	_, err := New(srv.URL, nil).Create(context.Background(), "", "x", nil)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes/abc" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeEnvelope(t, w, http.StatusOK, Note{ID: "abc", Title: "Go"}, nil)
	}))
	defer srv.Close()

	n, err := New(srv.URL, nil).Get(context.Background(), "abc")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if n.ID != "abc" || n.Title != "Go" {
		t.Fatalf("unexpected note: %+v", n)
	}
}

func TestGet_NotFoundMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusNotFound, nil, &apiError{Code: "not_found", Message: "note not found"})
	}))
	defer srv.Close()

	_, err := New(srv.URL, nil).Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_IDIsPathEscaped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v1/notes/a%2Fb" {
			t.Fatalf("expected escaped id in path, got %s", r.URL.EscapedPath())
		}
		writeEnvelope(t, w, http.StatusOK, Note{ID: "a/b"}, nil)
	}))
	defer srv.Close()

	if _, err := New(srv.URL, nil).Get(context.Background(), "a/b"); err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
}

func TestListRecent_UsesLimitAndOffsetZero(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "5" || r.URL.Query().Get("offset") != "0" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []Note{{ID: "1"}, {ID: "2"}},
		}, nil)
	}))
	defer srv.Close()

	notes, err := New(srv.URL, nil).ListRecent(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListRecent: unexpected error: %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
}

func TestSearch_CallsSearchEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" || r.URL.Query().Get("q") != "goroutines pool" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []Note{{ID: "1", Title: "Go"}},
		}, nil)
	}))
	defer srv.Close()

	results, err := New(srv.URL, nil).Search(context.Background(), "goroutines pool", 10)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "1" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestSearch_TruncatesToLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []Note{{ID: "1"}, {ID: "2"}, {ID: "3"}},
		}, nil)
	}))
	defer srv.Close()

	results, err := New(srv.URL, nil).Search(context.Background(), "go", 2)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected results truncated to 2, got %d", len(results))
	}
}

func TestSearch_NoTruncationWhenLimitZeroOrLarger(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []Note{{ID: "1"}, {ID: "2"}},
		}, nil)
	}))
	defer srv.Close()

	results, err := New(srv.URL, nil).Search(context.Background(), "go", 0)
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected no truncation with limit=0, got %d", len(results))
	}
}

func TestListRecent_PropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusInternalServerError, nil, &apiError{Code: "internal_error", Message: "boom"})
	}))
	defer srv.Close()

	if _, err := New(srv.URL, nil).ListRecent(context.Background(), 5); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestSearch_PropagatesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusInternalServerError, nil, &apiError{Code: "internal_error", Message: "boom"})
	}))
	defer srv.Close()

	if _, err := New(srv.URL, nil).Search(context.Background(), "go", 5); err == nil {
		t.Fatalf("expected an error to propagate")
	}
}

func TestDo_GenericAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusInternalServerError, nil, &apiError{Code: "internal_error", Message: "boom"})
	}))
	defer srv.Close()

	_, err := New(srv.URL, nil).Get(context.Background(), "1")
	if err == nil || errors.Is(err, ErrNotFound) || errors.Is(err, ErrValidation) {
		t.Fatalf("expected a generic (non-sentinel) error, got %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error message to include server message, got %v", err)
	}
}

func TestDo_MissingErrorBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusInternalServerError, nil, nil)
	}))
	defer srv.Close()

	_, err := New(srv.URL, nil).Get(context.Background(), "1")
	if err == nil {
		t.Fatalf("expected an error when the envelope has no error body")
	}
}

func TestDo_MalformedResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not json"))
	}))
	defer srv.Close()

	if _, err := New(srv.URL, nil).Get(context.Background(), "1"); err == nil {
		t.Fatalf("expected a decode error for a malformed response body")
	}
}

func TestDo_Unreachable(t *testing.T) {
	c := New("http://127.0.0.1:1", nil) // nothing listens here
	if _, err := c.Get(context.Background(), "1"); err == nil {
		t.Fatalf("expected an error when the API is unreachable")
	}
}

func TestDo_PerCallTimeoutIsEnforced(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		writeEnvelope(t, w, http.StatusOK, Note{ID: "1"}, nil)
	}))
	defer srv.Close()

	c := New(srv.URL, nil)
	c.Timeout = 5 * time.Millisecond
	_, err := c.Get(context.Background(), "1")
	if err == nil {
		t.Fatalf("expected a timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestBaseURL_TrailingSlashTrimmed(t *testing.T) {
	c := New("http://example.com/", nil)
	if c.baseURL != "http://example.com" {
		t.Fatalf("expected trailing slash trimmed, got %q", c.baseURL)
	}
}
