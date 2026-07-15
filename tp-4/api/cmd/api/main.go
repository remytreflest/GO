// Command api starts the Mira notes HTTP API.
//
//	@title			Mira Notes API
//	@version		2.0
//	@description	API v1 pour la gestion de notes du projet fil rouge Mira, avec stockage PostgreSQL, enrichissement asynchrone (tags/résumé/score/embedding simulés) et recherche hybride full-text + vectorielle.
//	@host			localhost:8080
//	@BasePath		/
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "mira/tp-4/api/docs"
	"mira/tp-4/api/internal/core"
	"mira/tp-4/api/internal/enrichment"
	"mira/tp-4/api/internal/http/handlers"
	"mira/tp-4/api/internal/http/middleware"
	"mira/tp-4/api/internal/store/postgres"
)

const (
	requestTimeout      = 5 * time.Second
	shutdownGracePeriod = 10 * time.Second

	defaultEnrichmentWorkers   = 4
	defaultEnrichmentQueueSize = 128
	defaultEnrichmentTimeout   = 3 * time.Second
)

// buildHandler wires the store, the enricher, the /api/v1 router, the
// Swagger UI and the middleware chain together. Store and enricher are
// injected so tests can exercise the full chain offline (in-memory store,
// fake enricher) without a real Postgres or worker pool.
func buildHandler(logger *slog.Logger, store core.Store, enricher handlers.Enricher) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/", handlers.NewRouter(store, enricher))
	mux.Handle("/swagger/", httpSwagger.WrapHandler)

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logging(logger),
		middleware.Recovery(logger),
		middleware.Timeout(requestTimeout),
	)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := resolveDSN()

	if err := runMigrations(dsn, logger); err != nil {
		logger.Error("migrations_failed", "error", err)
		os.Exit(1)
	}

	pgStore, err := postgres.New(ctx, dsn)
	if err != nil {
		logger.Error("postgres_connect_failed", "error", err)
		os.Exit(1)
	}
	defer pgStore.Close()

	pool := enrichment.NewPool(
		pgStore,
		getenvInt("ENRICHMENT_WORKERS", defaultEnrichmentWorkers),
		getenvInt("ENRICHMENT_QUEUE_SIZE", defaultEnrichmentQueueSize),
		getenvDuration("ENRICHMENT_JOB_TIMEOUT", defaultEnrichmentTimeout),
		logger,
	)
	pool.Start(ctx)

	srv := &http.Server{
		Addr:         resolveAddr(),
		Handler:      buildHandler(logger, pgStore, pool),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server_starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting_down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()

	// Order matters: stop accepting new requests (and therefore new
	// enrichment submissions) before draining the pool, and keep the DB
	// pool open until the pool has finished using it.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server_shutdown_error", "error", err)
	}
	if err := pool.Shutdown(shutdownCtx); err != nil {
		logger.Error("enrichment_pool_shutdown_error", "error", err)
	}
}
