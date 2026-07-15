package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/jackc/pgx/v5"

	"mira/tp-4/api/internal/core"
)

// scanner is satisfied by both pgx.Row and pgx.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanNote(row scanner) (*core.Note, error) {
	var n core.Note
	var status string
	var tags []string
	if err := row.Scan(
		&n.ID, &n.Title, &n.Content, &n.Summary, &n.Score, &status,
		&n.CreatedAt, &n.UpdatedAt, &tags,
	); err != nil {
		return nil, err
	}
	n.EnrichmentStatus = core.EnrichmentStatus(status)
	n.Tags = tags
	return &n, nil
}

const noteWithTagsQuery = `
	SELECT n.id, n.title, n.content, n.summary, n.score, n.enrichment_status,
	       n.created_at, n.updated_at,
	       COALESCE(array_agg(t.tag) FILTER (WHERE t.tag IS NOT NULL), '{}')
	FROM notes n
	LEFT JOIN note_tags t ON t.note_id = n.id
`

func (s *Store) Create(ctx context.Context, n *core.Note) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO notes (id, title, content, enrichment_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, n.ID, n.Title, n.Content, string(n.EnrichmentStatus), n.CreatedAt, n.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert note: %w", err)
	}

	if len(n.Tags) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO note_tags (note_id, tag) SELECT $1, unnest($2::text[])
		`, n.ID, n.Tags); err != nil {
			return fmt.Errorf("insert tags: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) Get(ctx context.Context, id string) (*core.Note, error) {
	row := s.pool.QueryRow(ctx, noteWithTagsQuery+" WHERE n.id = $1 GROUP BY n.id", id)
	n, err := scanNote(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrNotFound
		}
		return nil, fmt.Errorf("get note: %w", err)
	}
	return n, nil
}

func (s *Store) List(ctx context.Context, limit, offset int) ([]*core.Note, int, error) {
	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM notes`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notes: %w", err)
	}

	effectiveLimit := limit
	if effectiveLimit <= 0 {
		effectiveLimit = math.MaxInt32
	}

	rows, err := s.pool.Query(ctx, noteWithTagsQuery+`
		GROUP BY n.id
		ORDER BY n.created_at DESC
		LIMIT $1 OFFSET $2
	`, effectiveLimit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	notes := make([]*core.Note, 0)
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan note: %w", err)
		}
		notes = append(notes, n)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate notes: %w", err)
	}
	return notes, total, nil
}

func (s *Store) Update(ctx context.Context, id string, patch core.UpdateNoteInput) (*core.Note, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Any patch invalidates the previous enrichment: it must run again.
	setParts := []string{"updated_at = now()", "enrichment_status = 'pending'"}
	args := []any{}
	argN := 1
	if patch.Title != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argN))
		args = append(args, strings.TrimSpace(*patch.Title))
		argN++
	}
	if patch.Content != nil {
		setParts = append(setParts, fmt.Sprintf("content = $%d", argN))
		args = append(args, *patch.Content)
		argN++
	}
	args = append(args, id)

	tag, err := tx.Exec(ctx, fmt.Sprintf(
		"UPDATE notes SET %s WHERE id = $%d", strings.Join(setParts, ", "), argN,
	), args...)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, core.ErrNotFound
	}

	if patch.Tags != nil {
		if _, err := tx.Exec(ctx, `DELETE FROM note_tags WHERE note_id = $1`, id); err != nil {
			return nil, fmt.Errorf("clear tags: %w", err)
		}
		if len(*patch.Tags) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO note_tags (note_id, tag) SELECT $1, unnest($2::text[])
			`, id, *patch.Tags); err != nil {
				return nil, fmt.Errorf("insert tags: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM notes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return core.ErrNotFound
	}
	return nil
}
