package postgres

import "github.com/pgvector/pgvector-go"

// encodeVector returns the pgvector text literal for embedding (e.g.
// "[0.1,0.2,...]"), to be bound as a string query parameter alongside an
// explicit ::vector cast in the SQL text. pgvector-go v0.4.0 only implements
// database/sql's Scanner/Valuer, not a native pgx v5 binary codec, so this
// text-literal + cast approach is used instead of pgx type registration.
func encodeVector(embedding []float32) string {
	return pgvector.NewVector(embedding).String()
}
