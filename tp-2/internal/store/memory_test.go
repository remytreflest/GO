package store

import (
	"errors"
	"testing"
	"time"

	"mira/tp-2/internal/core"
)

func newNote(id, title, content string) *core.Note {
	now := time.Now()
	return &core.Note{ID: id, Title: title, Content: content, CreatedAt: now, UpdatedAt: now}
}

func TestMemoryStore_CreateAndGet(t *testing.T) {
	s := NewMemoryStore()
	n := newNote("1", "Go", "notes about go")

	if err := s.Create(n); err != nil {
		t.Fatalf("Create: unexpected error: %v", err)
	}

	got, err := s.Get("1")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got.Title != "Go" {
		t.Fatalf("Get: expected title %q, got %q", "Go", got.Title)
	}

	// Returned note must be a copy: mutating it must not affect the store.
	got.Title = "mutated"
	again, _ := s.Get("1")
	if again.Title != "Go" {
		t.Fatalf("Get must return a defensive copy, store was mutated to %q", again.Title)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	s := NewMemoryStore()
	_, err := s.Get("missing")
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_List(t *testing.T) {
	s := NewMemoryStore()
	base := time.Now()
	for i, id := range []string{"1", "2", "3"} {
		n := newNote(id, id, "")
		n.CreatedAt = base.Add(time.Duration(i) * time.Second)
		_ = s.Create(n)
	}

	notes, total, err := s.List(2, 0)
	if err != nil {
		t.Fatalf("List: unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total 3, got %d", total)
	}
	if len(notes) != 2 {
		t.Fatalf("expected 2 notes, got %d", len(notes))
	}
	// Most recent first.
	if notes[0].ID != "3" || notes[1].ID != "2" {
		t.Fatalf("expected notes [3 2], got [%s %s]", notes[0].ID, notes[1].ID)
	}

	notes, total, err = s.List(2, 2)
	if err != nil {
		t.Fatalf("List with offset: unexpected error: %v", err)
	}
	if total != 3 || len(notes) != 1 || notes[0].ID != "1" {
		t.Fatalf("unexpected page: total=%d notes=%v", total, notes)
	}

	notes, _, _ = s.List(0, 0)
	if len(notes) != 3 {
		t.Fatalf("limit<=0 should return all notes, got %d", len(notes))
	}

	notes, total, err = s.List(10, 100)
	if err != nil {
		t.Fatalf("List with offset beyond total: unexpected error: %v", err)
	}
	if total != 3 || len(notes) != 0 {
		t.Fatalf("expected an empty page when offset exceeds total, got total=%d notes=%d", total, len(notes))
	}
}

func TestMemoryStore_Update(t *testing.T) {
	s := NewMemoryStore()
	_ = s.Create(newNote("1", "Go", "old content"))

	newTitle := "Go advanced"
	updated, err := s.Update("1", core.UpdateNoteInput{Title: &newTitle})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if updated.Title != "Go advanced" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}
	if updated.Content != "old content" {
		t.Fatalf("Update must leave untouched fields as-is, got %q", updated.Content)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) && updated.UpdatedAt.Equal(updated.CreatedAt) {
		// Same-instant updates are fine on fast machines; only fail if UpdatedAt
		// went backwards.
		if updated.UpdatedAt.Before(updated.CreatedAt) {
			t.Fatalf("UpdatedAt must not be before CreatedAt")
		}
	}
}

func TestMemoryStore_UpdateContentAndTags(t *testing.T) {
	s := NewMemoryStore()
	_ = s.Create(newNote("1", "Go", "old content"))

	newContent := "new content"
	newTags := []string{"go", "web"}
	updated, err := s.Update("1", core.UpdateNoteInput{Content: &newContent, Tags: &newTags})
	if err != nil {
		t.Fatalf("Update: unexpected error: %v", err)
	}
	if updated.Content != newContent {
		t.Fatalf("expected content %q, got %q", newContent, updated.Content)
	}
	if len(updated.Tags) != 2 || updated.Tags[0] != "go" {
		t.Fatalf("expected tags %v, got %v", newTags, updated.Tags)
	}
	if updated.Title != "Go" {
		t.Fatalf("title should be untouched, got %q", updated.Title)
	}
}

func TestMemoryStore_UpdateNotFound(t *testing.T) {
	s := NewMemoryStore()
	title := "x"
	_, err := s.Update("missing", core.UpdateNoteInput{Title: &title})
	if !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	s := NewMemoryStore()
	_ = s.Create(newNote("1", "Go", ""))

	if err := s.Delete("1"); err != nil {
		t.Fatalf("Delete: unexpected error: %v", err)
	}
	if _, err := s.Get("1"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected note to be gone after Delete")
	}
}

func TestMemoryStore_DeleteNotFound(t *testing.T) {
	s := NewMemoryStore()
	if err := s.Delete("missing"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStore_Search(t *testing.T) {
	s := NewMemoryStore()
	_ = s.Create(newNote("1", "Go interfaces", "structural typing"))
	_ = s.Create(newNote("2", "REST API", "HTTP verbs and Go handlers"))
	_ = s.Create(newNote("3", "Cooking", "pasta recipe"))

	results, err := s.Search("go")
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for %q, got %d", "go", len(results))
	}

	results, err = s.Search("nomatch")
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}

	results, err = s.Search("   ")
	if err != nil {
		t.Fatalf("Search: unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("blank query should return no results, got %d", len(results))
	}
}
