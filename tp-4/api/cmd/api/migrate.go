package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver

	"mira/tp-4/api/migrations"
)

// runMigrations applies every pending embedded SQL migration. It uses a
// short-lived database/sql connection (golang-migrate's Postgres driver
// operates over database/sql, not pgxpool) that is closed before the
// application opens its long-lived pgxpool.Pool.
func runMigrations(dsn string, logger *slog.Logger) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migration connection: %w", err)
	}
	defer db.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("open embedded migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Info("migrations_up_to_date")
			return nil
		}
		return fmt.Errorf("apply migrations: %w", err)
	}
	logger.Info("migrations_applied")
	return nil
}
