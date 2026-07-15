package core

import "context"

// Store is the persistence contract for notes. Implementations own
// concurrency control so that Update can apply a patch atomically.
type Store interface {
	Create(ctx context.Context, n *Note) error
	Get(ctx context.Context, id string) (*Note, error)
	List(ctx context.Context, limit, offset int) (notes []*Note, total int, err error)
	Update(ctx context.Context, id string, patch UpdateNoteInput) (*Note, error)
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, query string) ([]*Note, error)
}
