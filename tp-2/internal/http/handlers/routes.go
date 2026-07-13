package handlers

import (
	"net/http"

	"mira/tp-2/internal/core"
)

// NewRouter builds the /api/v1 mux. It is deliberately dependency-free
// (stdlib net/http, Go 1.22+ method+pattern routing) so no external router
// is required.
func NewRouter(store core.Store) *http.ServeMux {
	h := NewNotesHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/notes", h.Create)
	mux.HandleFunc("GET /api/v1/notes", h.List)
	mux.HandleFunc("GET /api/v1/notes/{id}", h.Get)
	mux.HandleFunc("PATCH /api/v1/notes/{id}", h.Update)
	mux.HandleFunc("DELETE /api/v1/notes/{id}", h.Delete)
	mux.HandleFunc("GET /api/v1/search", h.Search)

	return mux
}
