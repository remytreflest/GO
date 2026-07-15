// Package postgres implements core.Store and core.EnrichmentStore backed by
// PostgreSQL (via pgx) with the pgvector extension for embedding storage and
// similarity search.
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"mira/tp-4/api/internal/core"
)

// Store is a PostgreSQL-backed implementation of core.Store and
// core.EnrichmentStore.
type Store struct {
	pool *pgxpool.Pool
}

var (
	_ core.Store           = (*Store)(nil)
	_ core.EnrichmentStore = (*Store)(nil)
)

// New opens a connection pool to dsn and verifies connectivity with a Ping.
//
// pgvector-go v0.4.0 only implements database/sql's Scanner/Valuer, not a
// native pgx v5 binary codec, so vector values are exchanged as their text
// literal form ("[0.1,0.2,...]") with an explicit ::vector cast in SQL
// rather than via pgx type registration — see notes.go/enrichment.go.
func New(ctx context.Context, dsn string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases all pooled connections.
func (s *Store) Close() {
	s.pool.Close()
}
