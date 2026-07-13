package handlers

import (
	"bytes"
	"net/http/httptest"
	"testing"
)

func TestDecodeJSON_UnknownField(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"title":"x","bogus":1}`))
	var v struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(req, &v); err == nil {
		t.Fatalf("expected an error for an unknown field")
	}
}

func TestDecodeJSON_TrailingData(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"title":"x"}{"title":"y"}`))
	var v struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(req, &v); err == nil {
		t.Fatalf("expected an error for trailing JSON data")
	}
}

func TestDecodeJSON_Valid(t *testing.T) {
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"title":"x"}`))
	var v struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(req, &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Title != "x" {
		t.Fatalf("expected title %q, got %q", "x", v.Title)
	}
}
