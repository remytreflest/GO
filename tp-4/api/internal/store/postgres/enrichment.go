package postgres

import (
	"context"
	"fmt"

	"mira/tp-4/api/internal/core"
)

// SaveEnrichment applies the outcome of an async enrichment job. On success
// (core.EnrichmentDone) it overwrites the note's tags/summary/score and
// upserts its embedding; on failure it only flips enrichment_status so the
// note doesn't stay stuck at "pending".
func (s *Store) SaveEnrichment(ctx context.Context, result core.EnrichmentResult) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin enrichment tx: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE notes SET enrichment_status = $1 WHERE id = $2`,
		string(result.Status), result.NoteID)
	if err != nil {
		return fmt.Errorf("update enrichment status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}

	if result.Status == core.EnrichmentDone {
		if _, err := tx.Exec(ctx, `UPDATE notes SET summary = $1, score = $2 WHERE id = $3`,
			result.Summary, result.Score, result.NoteID); err != nil {
			return fmt.Errorf("update summary/score: %w", err)
		}

		if _, err := tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, result.NoteID); err != nil {
			return fmt.Errorf("clear tags: %w", err)
		}
		if len(result.Tags) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO note_tags (note_id, tag) SELECT $1, unnest($2::text[])
			`, result.NoteID, result.Tags); err != nil {
				return fmt.Errorf("insert tags: %w", err)
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO note_embeddings (note_id, embedding, updated_at)
			VALUES ($1, $2::vector, now())
			ON CONFLICT (note_id) DO UPDATE SET embedding = EXCLUDED.embedding, updated_at = now()
		`, result.NoteID, encodeVector(result.Embedding)); err != nil {
			return fmt.Errorf("upsert embedding: %w", err)
		}
	}

	return tx.Commit(ctx)
}
