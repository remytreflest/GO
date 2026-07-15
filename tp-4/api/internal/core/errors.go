package core

import "errors"

// ErrNotFound is returned by Store implementations when a note does not exist.
var ErrNotFound = errors.New("note not found")

// ValidationError collects per-field validation failures so handlers can
// report all of them at once instead of failing fast on the first error.
type ValidationError struct {
	Fields map[string]string
}

func NewValidationError() *ValidationError {
	return &ValidationError{Fields: map[string]string{}}
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

func (e *ValidationError) Add(field, message string) {
	e.Fields[field] = message
}

func (e *ValidationError) HasErrors() bool {
	return len(e.Fields) > 0
}
