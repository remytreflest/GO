package embedding

// Score returns a heuristic "richness" score in [0,1] based on content
// length and vocabulary diversity — purely illustrative, not a quality
// judgment, just enough signal to demonstrate the enrichment pipeline
// writing back a computed value.
func Score(title, content string) float64 {
	tokens := tokenize(title + " " + content)
	if len(tokens) == 0 {
		return 0
	}

	unique := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		unique[t] = true
	}
	diversity := float64(len(unique)) / float64(len(tokens))

	length := float64(len(content)) / 1000
	if length > 1 {
		length = 1
	}

	score := 0.5*length + 0.5*diversity
	if score > 1 {
		score = 1
	}
	return score
}
