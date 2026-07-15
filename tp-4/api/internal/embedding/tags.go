package embedding

import "sort"

// stopwords is a small FR/EN list excluded from tag extraction.
var stopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "that": true, "this": true,
	"from": true, "have": true, "are": true, "was": true, "were": true, "not": true,
	"but": true, "you": true, "your": true, "can": true, "will": true, "into": true,
	"le": true, "la": true, "les": true, "de": true, "des": true, "du": true, "un": true,
	"une": true, "et": true, "est": true, "sont": true, "pour": true, "avec": true,
	"dans": true, "sur": true, "que": true, "qui": true, "ce": true, "ces": true,
	"son": true, "sa": true, "ses": true, "au": true, "aux": true, "par": true, "en": true,
}

// ExtractTags returns up to max frequent, non-stopword tokens from text,
// ordered by descending frequency (ties broken alphabetically for
// determinism, since Go map iteration order is randomized).
func ExtractTags(text string, max int) []string {
	counts := make(map[string]int)
	for _, tok := range tokenize(text) {
		if len(tok) <= 2 || stopwords[tok] {
			continue
		}
		counts[tok]++
	}

	type entry struct {
		token string
		count int
	}
	entries := make([]entry, 0, len(counts))
	for tok, c := range counts {
		entries = append(entries, entry{tok, c})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		return entries[i].token < entries[j].token
	})

	if max > len(entries) {
		max = len(entries)
	}
	tags := make([]string, max)
	for i := 0; i < max; i++ {
		tags[i] = entries[i].token
	}
	return tags
}
