// Command api starts the Mira notes HTTP API.
//
//	@title			Mira Notes API
//	@version		1.0
//	@description	API v1 pour la gestion de notes du projet fil rouge Mira.
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "mira/tp-2/docs"
	"mira/tp-2/internal/http/handlers"
	"mira/tp-2/internal/http/middleware"
	"mira/tp-2/internal/store"
)

const requestTimeout = 5 * time.Second

// buildHandler wires the in-memory store, the /api/v1 router, the Swagger UI
// and the middleware chain together. Split out from main so it can be
// exercised by tests without starting a real listener.
func buildHandler(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/", handlers.NewRouter(store.NewMemoryStore()))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logging(logger),
		middleware.Recovery(logger),
		middleware.Timeout(requestTimeout),
	)
}

// resolveAddr returns the listen address, defaulting to :8080 unless PORT
// is set.
func resolveAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8080"
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	srv := &http.Server{
		Addr:         resolveAddr(),
		Handler:      buildHandler(logger),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("server_starting", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server_failed", "error", err)
		os.Exit(1)
	}
}
