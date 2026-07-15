// Package apiclient implements notes.Store by calling the tp-4/api HTTP
// API instead of touching a local file, so that every note created or
// modified from the CLI goes through the server and triggers async
// enrichment.
package apiclient

import "encoding/json"

// envelope mirrors the API's stable response shape: exactly one of Data or
// Error is set.
type envelope struct {
	Data      json.RawMessage `json:"data"`
	Error     *apiError       `json:"error"`
	RequestID string          `json:"request_id"`
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}
