package main

import (
	"errors"
	"sync"
)

var (
	ErrDuplicate  = errors.New("note already exists")
	ErrNotFound   = errors.New("note not found")
	ErrValidation = errors.New("note title is required")
)

type NoteStore interface {
	Save(n *Note) error
	Get(id string) (*Note, error)
	All() []*Note
}

type MemoryStore struct {
	mu    sync.Mutex
	notes map[string]*Note
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{notes: make(map[string]*Note)}
}

func (s *MemoryStore) Save(n *Note) error {
	if n.Title == "" {
		return ErrValidation
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.notes[n.ID]; exists {
		return ErrDuplicate
	}
	s.notes[n.ID] = n
	return nil
}

func (s *MemoryStore) Get(id string) (*Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, ok := s.notes[id]
	if !ok {
		return nil, ErrNotFound
	}
	return n, nil
}

func (s *MemoryStore) All() []*Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]*Note, 0, len(s.notes))
	for _, n := range s.notes {
		result = append(result, n)
	}
	return result
}
