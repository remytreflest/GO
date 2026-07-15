package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	"database/sql"

	"mira/tp-4/api/internal/core"
	"mira/tp-4/api/internal/embedding"
	"mira/tp-4/api/migrations"
)

// requireDSN skips the test unless DATABASE_URL points at a live, disposable
// Postgres+pgvector instance (see tp-4/docker-compose.yml). These tests are
// never run by default CI — only locally/explicitly, since they need Docker.
func requireDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set; skipping Postgres integration test (see tp-4/commands.md)")
	}
	return dsn
}

// newTestStore applies migrations, opens a Store, and truncates all tables
// so each test starts from a clean slate.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := requireDSN(t)
	ctx := context.Background()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		t.Fatalf("migrate driver: %v", err)
	}
	src, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatalf("migrate source: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		t.Fatalf("migrate instance: %v", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatalf("migrate up: %v", err)
	}

	if _, err := db.Exec(`TRUNCATE notes, note_tags, note_embeddings CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	store, err := New(ctx, dsn)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func newTestNote(id, title, content string, tags []string) *core.Note {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return &core.Note{
		ID: id, Title: title, Content: content, Tags: tags,
		EnrichmentStatus: core.EnrichmentPending,
		CreatedAt:        now, UpdatedAt: now,
	}
}

func TestStore_CreateGetWithTags(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	n := newTestNote("1", "Go", "goroutines and channels", []string{"go", "concurrency"})
	if err := s.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := s.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Go" || got.EnrichmentStatus != core.EnrichmentPending {
		t.Fatalf("unexpected note: %+v", got)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %v", got.Tags)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if _, err := s.Get(ctx, "missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_ListPagination(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	for _, id := range []string{"1", "2", "3"} {
		if err := s.Create(ctx, newTestNote(id, id, "", nil)); err != nil {
			t.Fatalf("Create %s: %v", id, err)
		}
	}

	notes, total, err := s.List(ctx, 2, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 3 || len(notes) != 2 {
		t.Fatalf("expected total=3 len=2, got total=%d len=%d", total, len(notes))
	}
}

func TestStore_UpdateResetsEnrichmentStatus(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	n := newTestNote("1", "Go", "old", []string{"go"})
	if err := s.Create(ctx, n); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SaveEnrichment(ctx, core.EnrichmentResult{
		NoteID: "1", Status: core.EnrichmentDone, Summary: "old", Score: 0.1, Tags: []string{"go"},
		Embedding: make([]float32, 64),
	}); err != nil {
		t.Fatalf("SaveEnrichment: %v", err)
	}

	newContent := "new content"
	newTags := []string{"go", "postgres"}
	updated, err := s.Update(ctx, "1", core.UpdateNoteInput{Content: &newContent, Tags: &newTags})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Content != newContent {
		t.Fatalf("expected content updated, got %q", updated.Content)
	}
	if updated.EnrichmentStatus != core.EnrichmentPending {
		t.Fatalf("expected enrichment_status reset to pending, got %q", updated.EnrichmentStatus)
	}
	if len(updated.Tags) != 2 {
		t.Fatalf("expected tags replaced, got %v", updated.Tags)
	}
}

func TestStore_UpdateNotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)
	title := "x"
	if _, err := s.Update(ctx, "missing", core.UpdateNoteInput{Title: &title}); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.Create(ctx, newTestNote("1", "Go", "", nil)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.Delete(ctx, "1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "1"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected note gone, got %v", err)
	}
	if err := s.Delete(ctx, "1"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound deleting again, got %v", err)
	}
}

func TestStore_SaveEnrichment_FailedLeavesFieldsUntouched(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	if err := s.Create(ctx, newTestNote("1", "Go", "content", nil)); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := s.SaveEnrichment(ctx, core.EnrichmentResult{NoteID: "1", Status: core.EnrichmentFailed}); err != nil {
		t.Fatalf("SaveEnrichment: %v", err)
	}

	got, err := s.Get(ctx, "1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EnrichmentStatus != core.EnrichmentFailed {
		t.Fatalf("expected status failed, got %q", got.EnrichmentStatus)
	}
	if got.Summary != "" || got.Score != 0 {
		t.Fatalf("failed enrichment must not set summary/score, got %+v", got)
	}
}

func TestStore_SaveEnrichment_NotFound(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	err := s.SaveEnrichment(ctx, core.EnrichmentResult{NoteID: "missing", Status: core.EnrichmentDone})
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_HybridSearch(t *testing.T) {
	ctx := context.Background()
	s := newTestStore(t)

	notes := []struct {
		id, title, content string
	}{
		{"1", "Go concurrency", "goroutines channels worker pool"},
		{"2", "Go channels deep dive", "goroutines channels select timeout"},
		{"3", "Recette de pates", "carbonara pancetta oeufs parmesan"},
	}
	for _, n := range notes {
		if err := s.Create(ctx, newTestNote(n.id, n.title, n.content, nil)); err != nil {
			t.Fatalf("Create %s: %v", n.id, err)
		}
	}

	// Enrich note 3 too so all three have embeddings and the vector-only
	// branch of the WHERE clause is meaningfully exercised either way.
	for _, n := range notes {
		text := n.title + " " + n.content
		if err := s.SaveEnrichment(ctx, core.EnrichmentResult{
			NoteID: n.id, Status: core.EnrichmentDone,
			Summary: n.content, Score: 0.5,
			Embedding: embedding.Embed(text),
		}); err != nil {
			t.Fatalf("SaveEnrichment %s: %v", n.id, err)
		}
	}

	results, err := s.Search(ctx, "goroutine channels")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].ID == "3" {
		t.Fatalf("expected an unrelated pasta note to rank last, got it first: %+v", results)
	}
}
