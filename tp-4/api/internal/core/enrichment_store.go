package core

import "context"

// EnrichmentResult is the outcome of an async enrichment job for one note.
// Status is EnrichmentFailed when the job errored or timed out; in that case
// Tags/Summary/Score/Embedding are left at their zero values.
type EnrichmentResult struct {
	NoteID    string
	Tags      []string
	Summary   string
	Score     float64
	Embedding []float32
	Status    EnrichmentStatus
}

// EnrichmentStore persists the result of an async enrichment job. It is kept
// separate from Store so that the generic note repository contract isn't
// coupled to the enrichment feature.
type EnrichmentStore interface {
	SaveEnrichment(ctx context.Context, result EnrichmentResult) error
}
