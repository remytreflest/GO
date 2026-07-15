package postgres

import (
	"context"
	"fmt"

	"mira/tp-4/api/internal/core"
	"mira/tp-4/api/internal/embedding"
)

// Ranking weights and thresholds for the hybrid search below. Named
// constants so the formula in the SQL is easy to read and tune.
const (
	textRankWeight      = 0.6
	vectorSimWeight     = 0.4
	minVectorSimilarity = 0.2 // notes below this cosine similarity don't match on vector alone
	searchResultLimit   = 20
)

// Search ranks notes by a weighted combination of full-text rank
// (ts_rank over the generated tsvector column) and vector cosine similarity
// (pgvector's <=> operator over embeddings produced by the enrichment
// pipeline). A note matches if either signal is strong enough: it contains
// the query's words, or its embedding is similar to the query's.
func (s *Store) Search(ctx context.Context, query string) ([]*core.Note, error) {
	queryEmbedding := encodeVector(embedding.Embed(query))

	rows, err := s.pool.Query(ctx, `
		SELECT n.id, n.title, n.content, n.summary, n.score, n.enrichment_status,
		       n.created_at, n.updated_at,
		       COALESCE(array_agg(t.tag) FILTER (WHERE t.tag IS NOT NULL), '{}')
		FROM notes n
		LEFT JOIN note_tags t ON t.note_id = n.id
		LEFT JOIN note_embeddings e ON e.note_id = n.id
		WHERE n.search_vector @@ plainto_tsquery('simple', $1)
		   OR (e.embedding IS NOT NULL AND 1 - (e.embedding <=> $2::vector) > $3)
		GROUP BY n.id, e.embedding
		ORDER BY ($4 * ts_rank(n.search_vector, plainto_tsquery('simple', $1))
		        + $5 * COALESCE(1 - (e.embedding <=> $2::vector), 0)) DESC
		LIMIT $6
	`, query, queryEmbedding, minVectorSimilarity, textRankWeight, vectorSimWeight, searchResultLimit)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	defer rows.Close()

	notes := make([]*core.Note, 0)
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return notes, nil
}
