package search

import (
	"strings"

	"mira/tp-1/internal/notes"
)

func Filter(all []*notes.Note, query string) []*notes.Note {
	q := strings.ToLower(query)
	var result []*notes.Note
	for _, n := range all {
		if strings.Contains(strings.ToLower(n.Title), q) ||
			strings.Contains(strings.ToLower(n.Content), q) {
			result = append(result, n)
		}
	}
	return result
}
