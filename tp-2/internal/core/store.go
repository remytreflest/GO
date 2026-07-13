package core

// Store is the persistence contract for notes. Implementations own
// concurrency control so that Update can apply a patch atomically.
type Store interface {
	Create(n *Note) error
	Get(id string) (*Note, error)
	List(limit, offset int) (notes []*Note, total int, err error)
	Update(id string, patch UpdateNoteInput) (*Note, error)
	Delete(id string) error
	Search(query string) ([]*Note, error)
}
