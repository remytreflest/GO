package main

import (
	"errors"
	"testing"
)

func TestSave_valid(t *testing.T) {
	store := NewMemoryStore()
	n := NewNote("1", "My Note", "Some content")
	if err := store.Save(n); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSave_emptyTitle(t *testing.T) {
	store := NewMemoryStore()
	n := &Note{ID: "1"}
	if err := store.Save(n); !errors.Is(err, ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestSave_duplicate(t *testing.T) {
	store := NewMemoryStore()
	n := NewNote("1", "My Note", "Content")
	_ = store.Save(n)
	if err := store.Save(n); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestGet_notFound(t *testing.T) {
	store := NewMemoryStore()
	if _, err := store.Get("?"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
