package main

import (
	"testing"
	"time"
)

func TestResolveDSN_Default(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if got := resolveDSN(); got != defaultDSN {
		t.Fatalf("expected default DSN, got %q", got)
	}
}

func TestResolveDSN_FromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x/y")
	if got := resolveDSN(); got != "postgres://x/y" {
		t.Fatalf("expected env DSN, got %q", got)
	}
}

func TestGetenvInt(t *testing.T) {
	t.Setenv("TEST_INT", "")
	if got := getenvInt("TEST_INT", 4); got != 4 {
		t.Fatalf("expected default 4, got %d", got)
	}
	t.Setenv("TEST_INT", "7")
	if got := getenvInt("TEST_INT", 4); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
	t.Setenv("TEST_INT", "not-a-number")
	if got := getenvInt("TEST_INT", 4); got != 4 {
		t.Fatalf("expected fallback to default on invalid int, got %d", got)
	}
}

func TestGetenvDuration(t *testing.T) {
	t.Setenv("TEST_DURATION", "")
	if got := getenvDuration("TEST_DURATION", 3*time.Second); got != 3*time.Second {
		t.Fatalf("expected default 3s, got %v", got)
	}
	t.Setenv("TEST_DURATION", "500ms")
	if got := getenvDuration("TEST_DURATION", 3*time.Second); got != 500*time.Millisecond {
		t.Fatalf("expected 500ms, got %v", got)
	}
	t.Setenv("TEST_DURATION", "bogus")
	if got := getenvDuration("TEST_DURATION", 3*time.Second); got != 3*time.Second {
		t.Fatalf("expected fallback to default on invalid duration, got %v", got)
	}
}
