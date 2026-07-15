package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"mira/tp-4/api/internal/enrichment"
	"mira/tp-4/api/internal/store"
)

// fakeEnricher discards jobs; buildHandler only needs something satisfying
// handlers.Enricher for these offline tests.
type fakeEnricher struct{}

func (fakeEnricher) Submit(enrichment.Job) bool { return true }

func TestResolveAddr_Default(t *testing.T) {
	t.Setenv("PORT", "")
	if got := resolveAddr(); got != ":8080" {
		t.Fatalf("expected default addr :8080, got %q", got)
	}
}

func TestResolveAddr_FromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	if got := resolveAddr(); got != ":9090" {
		t.Fatalf("expected addr :9090, got %q", got)
	}
}

func TestBuildHandler_ServesNotesAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := buildHandler(logger, store.NewMemoryStore(), fakeEnricher{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// Empty body -> validation failure, but this proves the full chain
	// (middlewares + router + handlers + store + enricher) is wired up.
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d (body=%s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected RequestID middleware to set X-Request-ID header")
	}
}
