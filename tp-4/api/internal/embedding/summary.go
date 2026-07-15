package embedding

import "strings"

// Summarize returns a short summary of text: its first sentence if that fits
// within maxLen, otherwise a hard truncation at a word boundary with an
// ellipsis.
func Summarize(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	if end := strings.IndexAny(text, ".!?"); end != -1 && end+1 <= maxLen {
		return strings.TrimSpace(text[:end+1])
	}

	if len(text) <= maxLen {
		return text
	}

	cut := strings.LastIndex(text[:maxLen], " ")
	if cut <= 0 {
		cut = maxLen
	}
	return strings.TrimSpace(text[:cut]) + "…"
}
