package notes

import "errors"

var (
	ErrNotFound   = errors.New("note not found")
	ErrDuplicate  = errors.New("note already exists")
	ErrValidation = errors.New("note title is required")
)

type Store interface {
	Save(n *Note) error
	Get(id string) (*Note, error)
	Delete(id string) error
	List() ([]*Note, error) // 10 dernières notes
	All() ([]*Note, error)  // toutes les notes (pour search)
}
