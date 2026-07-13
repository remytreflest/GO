package notes

import (
	"fmt"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

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
