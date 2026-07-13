package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"mira/tp-2/internal/core"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// NotesHandler exposes the /api/v1/notes and /api/v1/search endpoints.
type NotesHandler struct {
	Store core.Store
}

func NewNotesHandler(store core.Store) *NotesHandler {
	return &NotesHandler{Store: store}
}

type listResponse struct {
	Notes  []*core.Note `json:"notes"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type searchResponse struct {
	Notes []*core.Note `json:"notes"`
	Total int          `json:"total"`
	Query string       `json:"query"`
}

// Create handles POST /api/v1/notes.
//
//	@Summary		Créer une note
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Param			note	body		core.CreateNoteInput	true	"Note à créer"
//	@Success		201		{object}	envelope{data=core.Note}
//	@Failure		400		{object}	envelope{error=errorBody}
//	@Router			/api/v1/notes [post]
func (h *NotesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in core.CreateNoteInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	if ve := core.ValidateCreate(in); ve != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "invalid note payload", ve.Fields)
		return
	}

	now := time.Now()
	n := &core.Note{
		ID:        core.NewID(),
		Title:     strings.TrimSpace(in.Title),
		Content:   in.Content,
		Tags:      in.Tags,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.Store.Create(n); err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to create note", nil)
		return
	}
	writeJSON(w, r, http.StatusCreated, n)
}

// List handles GET /api/v1/notes.
//
//	@Summary		Lister les notes
//	@Tags			notes
//	@Produce		json
//	@Param			limit	query		int	false	"Nombre maximum de résultats (défaut 20, max 100)"
//	@Param			offset	query		int	false	"Décalage de pagination (défaut 0)"
//	@Success		200		{object}	envelope{data=listResponse}
//	@Failure		400		{object}	envelope{error=errorBody}
//	@Router			/api/v1/notes [get]
func (h *NotesHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_query", err.Error(), nil)
		return
	}

	notes, total, err := h.Store.List(limit, offset)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to list notes", nil)
		return
	}
	writeJSON(w, r, http.StatusOK, listResponse{Notes: notes, Total: total, Limit: limit, Offset: offset})
}

// Get handles GET /api/v1/notes/{id}.
//
//	@Summary		Récupérer une note
//	@Tags			notes
//	@Produce		json
//	@Param			id	path		string	true	"ID de la note"
//	@Success		200	{object}	envelope{data=core.Note}
//	@Failure		404	{object}	envelope{error=errorBody}
//	@Router			/api/v1/notes/{id} [get]
func (h *NotesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := h.Store.Get(id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "note not found", nil)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to get note", nil)
		return
	}
	writeJSON(w, r, http.StatusOK, n)
}

// Update handles PATCH /api/v1/notes/{id}.
//
//	@Summary		Mettre à jour partiellement une note
//	@Tags			notes
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"ID de la note"
//	@Param			patch	body		core.UpdateNoteInput	true	"Champs à modifier (au moins un requis)"
//	@Success		200		{object}	envelope{data=core.Note}
//	@Failure		400		{object}	envelope{error=errorBody}
//	@Failure		404		{object}	envelope{error=errorBody}
//	@Router			/api/v1/notes/{id} [patch]
func (h *NotesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var in core.UpdateNoteInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_body", err.Error(), nil)
		return
	}
	if in.Title == nil && in.Content == nil && in.Tags == nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "at least one field (title, content, tags) must be provided", nil)
		return
	}
	if ve := core.ValidateUpdate(in); ve != nil {
		writeError(w, r, http.StatusBadRequest, "validation_error", "invalid note payload", ve.Fields)
		return
	}

	n, err := h.Store.Update(id, in)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "note not found", nil)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to update note", nil)
		return
	}
	writeJSON(w, r, http.StatusOK, n)
}

// Delete handles DELETE /api/v1/notes/{id}.
//
//	@Summary		Supprimer une note
//	@Tags			notes
//	@Produce		json
//	@Param			id	path		string	true	"ID de la note"
//	@Success		200	{object}	envelope{data=map[string]string}
//	@Failure		404	{object}	envelope{error=errorBody}
//	@Router			/api/v1/notes/{id} [delete]
func (h *NotesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Store.Delete(id); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "note not found", nil)
			return
		}
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to delete note", nil)
		return
	}
	writeJSON(w, r, http.StatusOK, map[string]string{"id": id})
}

// Search handles GET /api/v1/search?q=...
//
//	@Summary		Rechercher des notes (titre + contenu)
//	@Tags			search
//	@Produce		json
//	@Param			q	query		string	true	"Terme recherché"
//	@Success		200	{object}	envelope{data=searchResponse}
//	@Failure		400	{object}	envelope{error=errorBody}
//	@Router			/api/v1/search [get]
func (h *NotesHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		writeError(w, r, http.StatusBadRequest, "invalid_query", "query parameter q is required", nil)
		return
	}

	results, err := h.Store.Search(q)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "internal_error", "failed to search notes", nil)
		return
	}
	writeJSON(w, r, http.StatusOK, searchResponse{Notes: results, Total: len(results), Query: q})
}

func parsePagination(r *http.Request) (limit, offset int, err error) {
	limit = defaultLimit
	q := r.URL.Query()

	if v := q.Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit < 0 {
			return 0, 0, errors.New("limit must be a non-negative integer")
		}
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	if v := q.Get("offset"); v != "" {
		offset, err = strconv.Atoi(v)
		if err != nil || offset < 0 {
			return 0, 0, errors.New("offset must be a non-negative integer")
		}
	}

	return limit, offset, nil
}
