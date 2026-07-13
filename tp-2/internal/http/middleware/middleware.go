// Package middleware provides HTTP middlewares: request ID, structured
// logging, panic recovery and request timeout.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestIDFromContext returns the request ID stored by the RequestID
// middleware, or "" if none is present.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// newRequestID generates a random ID. crypto/rand.Read does not fail on
// supported platforms, so its error is not handled.
func newRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestID assigns a request ID (reusing an incoming X-Request-ID header
// when present), stores it in the request context and echoes it back on
// the response so callers can correlate logs.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(status int) {
	rec.status = status
	rec.ResponseWriter.WriteHeader(status)
}

// Logging returns a middleware that emits one structured log entry per
// request via slog: method, path, status, duration and request ID.
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http_request",
				"request_id", RequestIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// Recovery returns a middleware that recovers from panics in the handler
// chain, logs them and responds with a 500 instead of crashing the server.
func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("panic_recovered",
						"request_id", RequestIDFromContext(r.Context()),
						"error", err,
					)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"internal server error"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout returns a middleware that aborts the request with a 503 once d
// elapses without a response being written.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, `{"error":{"code":"timeout","message":"request timed out"}}`)
	}
}

// Chain applies middlewares in order, so mw[0] is the outermost (runs first).
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
