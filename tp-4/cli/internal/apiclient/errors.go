package apiclient

import (
	"fmt"

	"mira/tp-4/cli/internal/notes"
)

// mapAPIError translates the API's {code, message} error body into the
// sentinel errors notes.Store callers already handle (notes.ErrNotFound,
// notes.ErrValidation), falling back to a generic wrapped error for
// anything else so the caller still gets a useful message.
func mapAPIError(status int, apiErr *apiError) error {
	if apiErr == nil {
		return fmt.Errorf("unexpected API response (status %d)", status)
	}
	switch apiErr.Code {
	case "not_found":
		return notes.ErrNotFound
	case "validation_error", "invalid_body", "invalid_query":
		return fmt.Errorf("%w: %s", notes.ErrValidation, apiErr.Message)
	default:
		return fmt.Errorf("api error %s: %s", apiErr.Code, apiErr.Message)
	}
}
