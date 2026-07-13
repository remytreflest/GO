// Package handlers implements the HTTP handlers for /api/v1/notes and
// /api/v1/search, and the router that wires them up.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"mira/tp-2/internal/http/middleware"
)

// envelope is the stable JSON shape returned by every endpoint: exactly one
// of data or error is set, alongside the request ID for correlation.
type envelope struct {
	Data      any        `json:"data,omitempty"`
	Error     *errorBody `json:"error,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		Data:      data,
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(envelope{
		Error:     &errorBody{Code: code, Message: message, Fields: fields},
		RequestID: middleware.RequestIDFromContext(r.Context()),
	})
}

// decodeJSON decodes a single JSON object from the request body, rejecting
// unknown fields, trailing data and empty bodies with a descriptive error.
func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body is required")
		}
		return errors.New("invalid JSON: " + err.Error())
	}
	if dec.More() {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
