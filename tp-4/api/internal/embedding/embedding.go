// Package embedding provides deterministic, dependency-free "enrichment"
// primitives used to simulate tags/summary/score/embedding generation
// without calling any external AI service. This keeps the pipeline fully
// offline and reproducible, at the cost of real semantic understanding:
// Embed clusters lexically similar notes together (shared words hash to the
// same buckets) but has no notion of synonyms or meaning.
package embedding

import (
	"hash/fnv"
	"math"
)

// Dimension is the fixed size of every embedding vector. It is hardcoded to
// match the `vector(64)` column in the note_embeddings migration — changing
// it requires changing the migration and resetting the table, since pgvector
// cannot resize a vector column in place.
const Dimension = 64

// sentinelToken is embedded in place of empty content so Embed never returns
// the zero vector (cosine distance against the zero vector is undefined).
const sentinelToken = "__empty__"

// Embed returns a deterministic, L2-normalized pseudo-semantic vector for
// text: a feature-hashed bag-of-words. Tokens are hashed into Dimension
// buckets and counted; the resulting vector is normalized to unit length so
// cosine similarity is well-defined and comparable across texts.
func Embed(text string) []float32 {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		tokens = []string{sentinelToken}
	}

	vec := make([]float64, Dimension)
	for _, tok := range tokens {
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		bucket := h.Sum32() % Dimension
		vec[bucket]++
	}

	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	out := make([]float32, Dimension)
	for i, v := range vec {
		out[i] = float32(v / norm)
	}
	return out
}
