package apiclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"mira/tp-4/cli/internal/notes"
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

func TestSave_AssignsServerIDAndFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/notes" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeEnvelope(t, w, http.StatusCreated, apiNote{
			ID: "server-id", Title: "Go", Content: "notes", EnrichmentStatus: "pending",
		}, nil)
	}))
	defer srv.Close()

	c := New(srv.URL)
	n := notes.New("Go", "notes")
	localID := n.ID

	if err := c.Save(n); err != nil {
		t.Fatalf("Save: unexpected error: %v", err)
	}
	if n.ID != "server-id" {
		t.Fatalf("expected server-assigned ID to overwrite local placeholder %q, got %q", localID, n.ID)
	}
	if n.EnrichmentStatus != "pending" {
		t.Fatalf("expected enrichment_status pending, got %q", n.EnrichmentStatus)
	}
}

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/notes/abc" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeEnvelope(t, w, http.StatusOK, apiNote{ID: "abc", Title: "Go"}, nil)
	}))
	defer srv.Close()

	n, err := New(srv.URL).Get("abc")
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

	_, err := New(srv.URL).Get("missing")
	if !errors.Is(err, notes.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSave_ValidationErrorMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusBadRequest, nil, &apiError{Code: "validation_error", Message: "title is required"})
	}))
	defer srv.Close()

	err := New(srv.URL).Save(notes.New("", "x"))
	if !errors.Is(err, notes.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]string{"id": "abc"}, nil)
	}))
	defer srv.Close()

	if err := New(srv.URL).Delete("abc"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
}

func TestList_UsesLimit10Offset0(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" || r.URL.Query().Get("offset") != "0" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []apiNote{{ID: "1"}, {ID: "2"}},
			"total": 2,
		}, nil)
	}))
	defer srv.Close()

	notesOut, err := New(srv.URL).List()
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if len(notesOut) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notesOut))
	}
}

func TestAll_PaginatesUntilTotalReached(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RawQuery)
		offset := r.URL.Query().Get("offset")

		var page []apiNote
		switch offset {
		case "0":
			for i := 0; i < 100; i++ {
				page = append(page, apiNote{ID: fmt.Sprintf("n%d", i)})
			}
		case "100":
			page = []apiNote{{ID: "n100"}, {ID: "n101"}}
		default:
			t.Fatalf("unexpected offset %q", offset)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]any{"notes": page, "total": 102}, nil)
	}))
	defer srv.Close()

	all, err := New(srv.URL).All()
	if err != nil {
		t.Fatalf("All: unexpected error: %v", err)
	}
	if len(all) != 102 {
		t.Fatalf("expected 102 notes across pages, got %d", len(all))
	}
	if len(requests) != 2 {
		t.Fatalf("expected exactly 2 page requests, got %d: %v", len(requests), requests)
	}
}

func TestAll_StopsOnEmptyPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// total lies (says 50) but the page is empty: must not loop forever.
		writeEnvelope(t, w, http.StatusOK, map[string]any{"notes": []apiNote{}, "total": 50}, nil)
	}))
	defer srv.Close()

	all, err := New(srv.URL).All()
	if err != nil {
		t.Fatalf("All: unexpected error: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected 0 notes, got %d", len(all))
	}
}

func TestSearch_CallsSearchEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/search" || r.URL.Query().Get("q") != "goroutines pool" {
			t.Fatalf("unexpected request: %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		writeEnvelope(t, w, http.StatusOK, map[string]any{
			"notes": []apiNote{{ID: "1", Title: "Go"}},
			"total": 1, "query": "goroutines pool",
		}, nil)
	}))
	defer srv.Close()

	results, err := New(srv.URL).Search("goroutines pool")
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "1" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestUpdate_SendsOnlyProvidedFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, hasContent := body["content"]; hasContent {
			t.Fatalf("expected content to be omitted, got body=%v", body)
		}
		if body["title"] != "Go updated" {
			t.Fatalf("expected title in body, got %v", body)
		}
		writeEnvelope(t, w, http.StatusOK, apiNote{ID: "1", Title: "Go updated", EnrichmentStatus: "pending"}, nil)
	}))
	defer srv.Close()

	title := "Go updated"
	updated, err := New(srv.URL).Update("1", &title, nil, nil)
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if updated.Title != "Go updated" || updated.EnrichmentStatus != "pending" {
		t.Fatalf("unexpected updated note: %+v", updated)
	}
}

func TestPing_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusOK, map[string]any{"notes": []apiNote{}, "total": 0}, nil)
	}))
	defer srv.Close()

	if err := New(srv.URL).Ping(); err != nil {
		t.Fatalf("Ping: unexpected error: %v", err)
	}
}

func TestPing_Unreachable(t *testing.T) {
	c := New("http://127.0.0.1:1") // nothing listens here
	if err := c.Ping(); err == nil {
		t.Fatalf("expected an error when the API is unreachable")
	}
}

func TestDo_GenericAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(t, w, http.StatusInternalServerError, nil, &apiError{Code: "internal_error", Message: "boom"})
	}))
	defer srv.Close()

	err := New(srv.URL).Delete("1")
	if err == nil || errors.Is(err, notes.ErrNotFound) || errors.Is(err, notes.ErrValidation) {
		t.Fatalf("expected a generic (non-sentinel) error, got %v", err)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected error message to include server message, got %v", err)
	}
}

func TestDo_MalformedResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not json"))
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Get("1"); err == nil {
		t.Fatalf("expected a decode error for a malformed response body")
	}
}

func TestBaseURL_TrailingSlashTrimmed(t *testing.T) {
	c := New("http://example.com/")
	if c.baseURL != "http://example.com" {
		t.Fatalf("expected trailing slash trimmed, got %q", c.baseURL)
	}
}
