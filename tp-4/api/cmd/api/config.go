package main

import (
	"os"
	"strconv"
	"time"
)

// resolveAddr returns the listen address, defaulting to :8080 unless PORT
// is set.
func resolveAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":8080"
}

const defaultDSN = "postgres://mira:mira@localhost:5433/mira?sslmode=disable"

// resolveDSN returns the Postgres connection string, defaulting to the
// docker-compose dev database unless DATABASE_URL is set.
func resolveDSN() string {
	return getenvString("DATABASE_URL", defaultDSN)
}

func getenvString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getenvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
