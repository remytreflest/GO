package core

import "time"

// EnrichmentStatus tracks the lifecycle of a note's async enrichment job.
type EnrichmentStatus string

const (
	EnrichmentPending EnrichmentStatus = "pending"
	EnrichmentDone    EnrichmentStatus = "done"
	EnrichmentFailed  EnrichmentStatus = "failed"
)

// Note is the domain entity for a note.
type Note struct {
	ID               string           `json:"id"`
	Title            string           `json:"title"`
	Content          string           `json:"content"`
	Tags             []string         `json:"tags,omitempty"`
	Summary          string           `json:"summary,omitempty"`
	Score            float64          `json:"score"`
	EnrichmentStatus EnrichmentStatus `json:"enrichment_status"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// CreateNoteInput is the payload accepted by POST /api/v1/notes.
type CreateNoteInput struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags,omitempty"`
}

// UpdateNoteInput is the payload accepted by PATCH /api/v1/notes/{id}.
// Pointers distinguish "field absent" from "field set to zero value" for partial updates.
type UpdateNoteInput struct {
	Title   *string   `json:"title,omitempty"`
	Content *string   `json:"content,omitempty"`
	Tags    *[]string `json:"tags,omitempty"`
}
