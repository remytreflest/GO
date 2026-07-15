// Package enrichment runs note enrichment (tags/summary/score/embedding)
// asynchronously on a bounded worker pool, so that note creation/update
// requests return immediately while the actual computation happens in the
// background and is written back to the store.
package enrichment

// Job describes one note to enrich.
type Job struct {
	NoteID  string
	Title   string
	Content string
}
