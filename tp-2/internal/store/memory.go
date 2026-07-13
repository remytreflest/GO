// Package store provides Store implementations for notes.
package store

import (
	"sort"
	"strings"
	"sync"
	"time"

	"mira/tp-2/internal/core"
)

// MemoryStore is an in-memory, concurrency-safe implementation of core.Store.
type MemoryStore struct {
	mu    sync.RWMutex
	notes map[string]*core.Note
}

var _ core.Store = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{notes: make(map[string]*core.Note)}
}

func (s *MemoryStore) Create(n *core.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notes[n.ID] = n
	return nil
}

func (s *MemoryStore) Get(id string) (*core.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	if !ok {
		return nil, core.ErrNotFound
	}
	cp := *n
	return &cp, nil
}

func (s *MemoryStore) List(limit, offset int) ([]*core.Note, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]*core.Note, 0, len(s.notes))
	for _, n := range s.notes {
		cp := *n
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return all[offset:end], total, nil
}

func (s *MemoryStore) Update(id string, patch core.UpdateNoteInput) (*core.Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, ok := s.notes[id]
	if !ok {
		return nil, core.ErrNotFound
	}
	if patch.Title != nil {
		n.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Content != nil {
		n.Content = *patch.Content
	}
	if patch.Tags != nil {
		n.Tags = *patch.Tags
	}
	n.UpdatedAt = time.Now()

	cp := *n
	return &cp, nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.notes[id]; !ok {
		return core.ErrNotFound
	}
	delete(s.notes, id)
	return nil
}

func (s *MemoryStore) Search(query string) ([]*core.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	results := make([]*core.Note, 0)
	if q == "" {
		return results, nil
	}
	for _, n := range s.notes {
		if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Content), q) {
			cp := *n
			results = append(results, &cp)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results, nil
}
