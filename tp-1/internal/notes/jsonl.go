package notes

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
)

type JSONLStore struct {
	path string
}

func NewJSONLStore() (*JSONLStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".mira")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return &JSONLStore{path: filepath.Join(dir, "notes.jsonl")}, nil
}

func (s *JSONLStore) load() ([]*Note, error) {
	f, err := os.Open(s.path)
	if os.IsNotExist(err) {
		return []*Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var result []*Note
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var n Note
		if err := json.Unmarshal(sc.Bytes(), &n); err != nil {
			return nil, err
		}
		result = append(result, &n)
	}
	return result, sc.Err()
}

func (s *JSONLStore) flush(notes []*Note) error {
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f) // Encode() ajoute \n automatiquement → format JSONL
	for _, n := range notes {
		if err := enc.Encode(n); err != nil {
			return err
		}
	}
	return nil
}

func (s *JSONLStore) Save(n *Note) error {
	if n.Title == "" {
		return ErrValidation
	}
	notes, err := s.load()
	if err != nil {
		return err
	}
	for _, existing := range notes {
		if existing.ID == n.ID {
			return ErrDuplicate
		}
	}
	return s.flush(append(notes, n))
}

func (s *JSONLStore) Get(id string) (*Note, error) {
	notes, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, n := range notes {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, ErrNotFound
}

func (s *JSONLStore) Delete(id string) error {
	notes, err := s.load()
	if err != nil {
		return err
	}
	filtered := make([]*Note, 0, len(notes))
	found := false
	for _, n := range notes {
		if n.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, n)
	}
	if !found {
		return ErrNotFound
	}
	return s.flush(filtered)
}

func (s *JSONLStore) List() ([]*Note, error) {
	notes, err := s.load()
	if err != nil {
		return nil, err
	}
	if len(notes) > 10 {
		notes = notes[len(notes)-10:]
	}
	return notes, nil
}

func (s *JSONLStore) All() ([]*Note, error) {
	return s.load()
}
