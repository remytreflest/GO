-- IDs stay the existing 16-hex-char scheme generated in Go (core.NewID(),
-- crypto/rand) rather than UUIDs, so no pgcrypto/gen_random_uuid() is needed.
CREATE TABLE notes (
    id                 TEXT PRIMARY KEY,
    title              TEXT NOT NULL,
    content            TEXT NOT NULL,
    summary            TEXT NOT NULL DEFAULT '',
    score              DOUBLE PRECISION NOT NULL DEFAULT 0,
    enrichment_status  TEXT NOT NULL DEFAULT 'pending'
                        CHECK (enrichment_status IN ('pending', 'done', 'failed')),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- 'simple' (not 'french'/'english'): a generated column's expression must
    -- be immutable, and 'simple' keeps behavior deterministic and
    -- stemming-free for a bilingual teaching corpus.
    search_vector      tsvector GENERATED ALWAYS AS (
                           to_tsvector('simple',
                               coalesce(title, '') || ' ' || coalesce(content, '') || ' ' || coalesce(summary, ''))
                       ) STORED
);

CREATE INDEX notes_search_vector_gin_idx ON notes USING GIN (search_vector);
CREATE INDEX notes_created_at_idx ON notes (created_at DESC);
