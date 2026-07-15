-- vector(64) must match internal/embedding.Dimension exactly: pgvector
-- cannot resize a vector column in place, so changing the Go constant
-- requires a new migration (and a reset of this table's data).
CREATE TABLE note_embeddings (
    note_id    TEXT PRIMARY KEY REFERENCES notes (id) ON DELETE CASCADE,
    embedding  vector(64) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- HNSW rather than ivfflat: ivfflat needs a `lists` parameter tuned to the
-- table size and degrades badly (and needs periodic REINDEX) on a small,
-- incrementally-growing table like this teaching project's. HNSW has no such
-- tuning parameter and works correctly from zero rows onward.
CREATE INDEX note_embeddings_hnsw_idx
    ON note_embeddings USING hnsw (embedding vector_cosine_ops);
