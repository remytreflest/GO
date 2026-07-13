package main

import (
	"encoding/json"
	"os"
)

type Note struct {
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Tags    []string `json:"tags"`
}

func NewNote(title, content string) *Note {
	return &Note{Title: title, Content: content}
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

func LoadFromFile(path string) ([]*Note, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var notes []*Note
	if err := json.NewDecoder(f).Decode(&notes); err != nil {
		return nil, err
	}
	return notes, nil
}
