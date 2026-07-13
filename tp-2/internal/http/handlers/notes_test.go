package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira/tp-2/internal/core"
	"mira/tp-2/internal/http/middleware"
	"mira/tp-2/internal/store"
)

// newTestRouter wraps the router with the RequestID middleware so tests can
// assert on the response envelope's request_id, just like the real server.
func newTestRouter() http.Handler {
	return middleware.RequestID(NewRouter(store.NewMemoryStore()))
}

func doRequest(t *testing.T, mux http.Handler, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeEnvelope(t *testing.T, rec *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response envelope: %v (body=%s)", err, rec.Body.String())
	}
	return env
}

func TestCreate_Success(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{
		Title:   "Go",
		Content: "notes about go",
		Tags:    []string{"go"},
	})

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error != nil {
		t.Fatalf("expected no error, got %+v", env.Error)
	}
	if env.RequestID == "" {
		t.Fatalf("expected request_id to be set")
	}

	data, ok := env.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected data to be an object, got %T", env.Data)
	}
	if data["id"] == "" || data["id"] == nil {
		t.Fatalf("expected a generated id, got %v", data["id"])
	}
	if data["title"] != "Go" {
		t.Fatalf("expected title %q, got %v", "Go", data["title"])
	}
}

func TestCreate_InvalidPayload(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Content: "no title"})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error == nil {
		t.Fatalf("expected an error body")
	}
	if env.Error.Code != "validation_error" {
		t.Fatalf("expected code validation_error, got %q", env.Error.Code)
	}
	if _, ok := env.Error.Fields["title"]; !ok {
		t.Fatalf("expected a title field error, got %v", env.Error.Fields)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	mux := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString("{not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error == nil || env.Error.Code != "invalid_body" {
		t.Fatalf("expected invalid_body error, got %+v", env.Error)
	}
}

func TestCreate_EmptyBody(t *testing.T) {
	mux := newTestRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", bytes.NewBufferString(""))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestGet_Success(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	rec := doRequest(t, mux, http.MethodGet, "/api/v1/notes/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Data.(map[string]any)["id"] != id {
		t.Fatalf("expected note %q, got %v", id, env.Data)
	}
}

func TestGet_NotFound(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/notes/does-not-exist", nil)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Error == nil || env.Error.Code != "not_found" {
		t.Fatalf("expected not_found error, got %+v", env.Error)
	}
}

func TestList_Success(t *testing.T) {
	mux := newTestRouter()
	for _, title := range []string{"A", "B", "C"} {
		doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: title})
	}

	rec := doRequest(t, mux, http.MethodGet, "/api/v1/notes?limit=2&offset=1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	data := env.Data.(map[string]any)
	if data["total"] != float64(3) {
		t.Fatalf("expected total 3, got %v", data["total"])
	}
	notes := data["notes"].([]any)
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes for limit=2, got %d", len(notes))
	}
}

func TestList_InvalidQuery(t *testing.T) {
	mux := newTestRouter()

	rec := doRequest(t, mux, http.MethodGet, "/api/v1/notes?limit=abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad limit, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodGet, "/api/v1/notes?offset=abc", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad offset, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodGet, "/api/v1/notes?offset=-1", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative offset, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestList_LimitClampedToMax(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/notes?limit=1000", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Data.(map[string]any)["limit"] != float64(maxLimit) {
		t.Fatalf("expected limit clamped to %d, got %v", maxLimit, env.Data.(map[string]any)["limit"])
	}
}

func TestUpdate_Success(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	rec := doRequest(t, mux, http.MethodPatch, "/api/v1/notes/"+id, map[string]string{"title": "Go updated"})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	if env.Data.(map[string]any)["title"] != "Go updated" {
		t.Fatalf("expected updated title, got %v", env.Data)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodPatch, "/api/v1/notes/missing", map[string]string{"title": "x"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestUpdate_EmptyPatch(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	rec := doRequest(t, mux, http.MethodPatch, "/api/v1/notes/"+id, map[string]string{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty patch, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/notes/"+id, bytes.NewBufferString("{not json"))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestUpdate_InvalidPayload(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	rec := doRequest(t, mux, http.MethodPatch, "/api/v1/notes/"+id, map[string]string{"title": "  "})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestDelete_Success(t *testing.T) {
	mux := newTestRouter()
	created := decodeEnvelope(t, doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go"}))
	id := created.Data.(map[string]any)["id"].(string)

	rec := doRequest(t, mux, http.MethodDelete, "/api/v1/notes/"+id, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}

	rec = doRequest(t, mux, http.MethodGet, "/api/v1/notes/"+id, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected note to be gone, got %d", rec.Code)
	}
}

func TestDelete_NotFound(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodDelete, "/api/v1/notes/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestSearch_Success(t *testing.T) {
	mux := newTestRouter()
	doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Go interfaces", Content: "typing"})
	doRequest(t, mux, http.MethodPost, "/api/v1/notes", core.CreateNoteInput{Title: "Cooking", Content: "pasta"})

	rec := doRequest(t, mux, http.MethodGet, "/api/v1/search?q=go", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	env := decodeEnvelope(t, rec)
	data := env.Data.(map[string]any)
	if data["total"] != float64(1) {
		t.Fatalf("expected 1 result, got %v", data["total"])
	}
}

func TestSearch_MissingQuery(t *testing.T) {
	mux := newTestRouter()
	rec := doRequest(t, mux, http.MethodGet, "/api/v1/search", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d (body=%s)", rec.Code, rec.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	mux := newTestRouter()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
