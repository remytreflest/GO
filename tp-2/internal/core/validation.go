package core

import "strings"

const (
	maxTitleLen   = 200
	maxContentLen = 10000
	maxTagLen     = 50
)

// ValidateCreate returns nil when in is valid, otherwise a *ValidationError
// with one entry per invalid field.
func ValidateCreate(in CreateNoteInput) *ValidationError {
	ve := NewValidationError()

	title := strings.TrimSpace(in.Title)
	switch {
	case title == "":
		ve.Add("title", "title is required")
	case len(title) > maxTitleLen:
		ve.Add("title", "title must be at most 200 characters")
	}

	if len(in.Content) > maxContentLen {
		ve.Add("content", "content must be at most 10000 characters")
	}

	if err := validateTags(in.Tags); err != "" {
		ve.Add("tags", err)
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// ValidateUpdate returns nil when in is valid, otherwise a *ValidationError
// with one entry per invalid field. Only fields that are present (non-nil)
// are validated, since PATCH is a partial update.
func ValidateUpdate(in UpdateNoteInput) *ValidationError {
	ve := NewValidationError()

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		switch {
		case title == "":
			ve.Add("title", "title cannot be empty")
		case len(title) > maxTitleLen:
			ve.Add("title", "title must be at most 200 characters")
		}
	}

	if in.Content != nil && len(*in.Content) > maxContentLen {
		ve.Add("content", "content must be at most 10000 characters")
	}

	if in.Tags != nil {
		if err := validateTags(*in.Tags); err != "" {
			ve.Add("tags", err)
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateTags(tags []string) string {
	for _, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			return "tags must not be empty"
		}
		if len(tag) > maxTagLen {
			return "each tag must be at most 50 characters"
		}
	}
	return ""
}
