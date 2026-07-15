package core

import "testing"

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name    string
		in      CreateNoteInput
		wantErr bool
		field   string
	}{
		{"valid", CreateNoteInput{Title: "Go", Content: "notes"}, false, ""},
		{"valid with tags", CreateNoteInput{Title: "Go", Tags: []string{"go", "web"}}, false, ""},
		{"empty title", CreateNoteInput{Title: "  ", Content: "x"}, true, "title"},
		{"missing title", CreateNoteInput{Content: "x"}, true, "title"},
		{"title too long", CreateNoteInput{Title: string(make([]byte, 201))}, true, "title"},
		{"content too long", CreateNoteInput{Title: "Go", Content: string(make([]byte, 10001))}, true, "content"},
		{"empty tag", CreateNoteInput{Title: "Go", Tags: []string{" "}}, true, "tags"},
		{"tag too long", CreateNoteInput{Title: "Go", Tags: []string{string(make([]byte, 51))}}, true, "tags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := ValidateCreate(tt.in)
			if tt.wantErr {
				if ve == nil {
					t.Fatalf("expected validation error, got nil")
				}
				if _, ok := ve.Fields[tt.field]; !ok {
					t.Fatalf("expected error on field %q, got %v", tt.field, ve.Fields)
				}
				return
			}
			if ve != nil {
				t.Fatalf("expected no error, got %v", ve.Fields)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	title := "New title"
	emptyTitle := "   "
	longTitle := string(make([]byte, 201))
	longContent := string(make([]byte, 10001))

	tests := []struct {
		name    string
		in      UpdateNoteInput
		wantErr bool
		field   string
	}{
		{"no fields is valid", UpdateNoteInput{}, false, ""},
		{"valid title", UpdateNoteInput{Title: &title}, false, ""},
		{"empty title", UpdateNoteInput{Title: &emptyTitle}, true, "title"},
		{"title too long", UpdateNoteInput{Title: &longTitle}, true, "title"},
		{"content too long", UpdateNoteInput{Content: &longContent}, true, "content"},
		{"empty tag", UpdateNoteInput{Tags: &[]string{""}}, true, "tags"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve := ValidateUpdate(tt.in)
			if tt.wantErr {
				if ve == nil {
					t.Fatalf("expected validation error, got nil")
				}
				if _, ok := ve.Fields[tt.field]; !ok {
					t.Fatalf("expected error on field %q, got %v", tt.field, ve.Fields)
				}
				return
			}
			if ve != nil {
				t.Fatalf("expected no error, got %v", ve.Fields)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	ve := NewValidationError()
	if ve.HasErrors() {
		t.Fatalf("fresh ValidationError should have no errors")
	}
	ve.Add("title", "title is required")
	if !ve.HasErrors() {
		t.Fatalf("expected HasErrors to be true after Add")
	}
	if ve.Error() != "validation failed" {
		t.Fatalf("unexpected Error() message: %q", ve.Error())
	}
}

func TestNewID(t *testing.T) {
	a := NewID()
	b := NewID()
	if a == "" || b == "" {
		t.Fatalf("NewID must not return an empty string")
	}
	if a == b {
		t.Fatalf("NewID must generate distinct ids, got %q twice", a)
	}
	if len(a) != 16 {
		t.Fatalf("expected 16 hex chars, got %d (%q)", len(a), a)
	}
}
