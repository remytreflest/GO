package embedding

import "strings"

// tokenize lowercases text and splits it into words, stripping punctuation.
func tokenize(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !('a' <= r && r <= 'z' || '0' <= r && r <= '9' || r > 127)
	})
	tokens := make([]string, 0, len(fields))
	for _, f := range fields {
		if f != "" {
			tokens = append(tokens, f)
		}
	}
	return tokens
}
