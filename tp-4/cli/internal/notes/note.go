package notes

import (
	"fmt"
	"time"
)

// Note mirrors the API's note shape. Summary/Score/EnrichmentStatus are
// filled in by the server (via the async enrichment pipeline) and are empty
// until enrichment runs, hence the omitempty tags.
type Note struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Tags             []string  `json:"tags,omitempty"`
	Summary          string    `json:"summary,omitempty"`
	Score            float64   `json:"score,omitempty"`
	EnrichmentStatus string    `json:"enrichment_status,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// New builds a client-side draft note. Its ID/CreatedAt are placeholders:
// once saved through apiclient, the server's own values overwrite them.
func New(title, content string) *Note {
	return &Note{
		ID:        fmt.Sprintf("%x", time.Now().UnixNano()),
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

func (n *Note) Preview() string {
	if len(n.Content) <= 80 {
		return n.Content
	}
	return n.Content[:80]
}

func (n *Note) AddTag(tag string) {
	for _, t := range n.Tags {
		if t == tag {
			return
		}
	}
	n.Tags = append(n.Tags, tag)
}
