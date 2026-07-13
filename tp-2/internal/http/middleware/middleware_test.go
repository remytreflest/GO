package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRequestID_GeneratesWhenAbsent(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	RequestID(next).ServeHTTP(rec, req)

	if seen == "" {
		t.Fatalf("expected a generated request id in context")
	}
	if rec.Header().Get("X-Request-ID") != seen {
		t.Fatalf("expected response header to echo request id %q, got %q", seen, rec.Header().Get("X-Request-ID"))
	}
}

func TestRequestID_ReusesIncomingHeader(t *testing.T) {
	var seen string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", "client-supplied-id")
	rec := httptest.NewRecorder()
	RequestID(next).ServeHTTP(rec, req)

	if seen != "client-supplied-id" {
		t.Fatalf("expected incoming request id to be reused, got %q", seen)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	if got := RequestIDFromContext(httptest.NewRequest(http.MethodGet, "/", nil).Context()); got != "" {
		t.Fatalf("expected empty request id for bare context, got %q", got)
	}
}

func TestLogging_RecordsStatus(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notes", nil)
	rec := httptest.NewRecorder()
	Logging(logger)(next).ServeHTTP(rec, req)

	out := buf.String()
	if !strings.Contains(out, "status=418") {
		t.Fatalf("expected log line to record status 418, got: %s", out)
	}
	if !strings.Contains(out, "method=GET") {
		t.Fatalf("expected log line to record method, got: %s", out)
	}
}

func TestRecovery_CatchesPanic(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Recovery(logger)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 after recovered panic, got %d", rec.Code)
	}
	if !strings.Contains(buf.String(), "panic_recovered") {
		t.Fatalf("expected panic to be logged, got: %s", buf.String())
	}
}

func TestRecovery_PassesThroughWhenNoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Recovery(logger)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestTimeout_Exceeded(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(50 * time.Millisecond):
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Timeout(5*time.Millisecond)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 on timeout, got %d", rec.Code)
	}
}

func TestTimeout_NotExceeded(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	Timeout(50*time.Millisecond)(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestChain_OrderAndExecution(t *testing.T) {
	var order []string
	mkMW := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-in")
				next.ServeHTTP(w, r)
				order = append(order, name+"-out")
			})
		}
	}

	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	h := Chain(final, mkMW("a"), mkMW("b"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"a-in", "b-in", "handler", "b-out", "a-out"}
	if len(order) != len(want) {
		t.Fatalf("expected order %v, got %v", want, order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("expected order %v, got %v", want, order)
		}
	}
}
