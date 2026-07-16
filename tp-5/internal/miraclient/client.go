// Package miraclient is a thin HTTP client over the tp-4/api Mira API. It is
// deliberately the only way the MCP server touches notes: going through the
// HTTP API (never the database) guarantees that every note created here also
// triggers the API's asynchronous enrichment pipeline, exactly like tp-4/cli.
package miraclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultTimeout bounds every underlying HTTP call made by the client, so a
// slow or unreachable API can never hang a tool call indefinitely.
const DefaultTimeout = 10 * time.Second

// Sentinel errors that tool handlers can match on to produce clean,
// user-facing messages instead of leaking raw API error bodies.
var (
	ErrNotFound   = errors.New("note not found")
	ErrValidation = errors.New("invalid note payload")
)

// Note mirrors the API's note representation (core.Note in tp-4/api).
type Note struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Tags             []string  `json:"tags,omitempty"`
	Summary          string    `json:"summary,omitempty"`
	Score            float64   `json:"score"`
	EnrichmentStatus string    `json:"enrichment_status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Client calls the mira HTTP API.
type Client struct {
	baseURL string
	http    *http.Client
	// Timeout bounds each underlying HTTP call. Defaults to DefaultTimeout.
	Timeout time.Duration
}

// New builds a Client targeting baseURL (e.g. "http://localhost:8080"). If
// httpClient is nil, a default one is used.
func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: httpClient}
}

// envelope mirrors the API's stable response shape: exactly one of Data or
// Error is set.
type envelope struct {
	Data  json.RawMessage `json:"data"`
	Error *apiError       `json:"error"`
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields"`
}

func (c *Client) timeout() time.Duration {
	if c.Timeout > 0 {
		return c.Timeout
	}
	return DefaultTimeout
}

// do sends an HTTP request bounded by a per-call timeout derived from ctx,
// decodes the {data,error} envelope, and unmarshals Data into out (if
// non-nil) on success.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout())
	defer cancel()

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call mira API at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response from mira API: %w", err)
	}

	if resp.StatusCode >= 400 {
		return mapAPIError(resp.StatusCode, env.Error)
	}
	if out != nil && len(env.Data) > 0 {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("decode data: %w", err)
		}
	}
	return nil
}

func mapAPIError(status int, apiErr *apiError) error {
	if apiErr == nil {
		return fmt.Errorf("unexpected mira API response (status %d)", status)
	}
	switch apiErr.Code {
	case "not_found":
		return ErrNotFound
	case "validation_error", "invalid_body", "invalid_query":
		return fmt.Errorf("%w: %s", ErrValidation, apiErr.Message)
	default:
		return fmt.Errorf("mira API error %s: %s", apiErr.Code, apiErr.Message)
	}
}

// Create creates a note via POST /api/v1/notes. The returned note carries
// the server-assigned ID and starts with enrichment_status "pending"; the
// API's async enrichment pipeline picks it up right after this call returns.
func (c *Client) Create(ctx context.Context, title, content string, tags []string) (*Note, error) {
	payload := struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags,omitempty"`
	}{Title: title, Content: content, Tags: tags}

	var created Note
	if err := c.do(ctx, http.MethodPost, "/api/v1/notes", payload, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// Get retrieves a single note by ID via GET /api/v1/notes/{id}.
func (c *Client) Get(ctx context.Context, id string) (*Note, error) {
	var n Note
	if err := c.do(ctx, http.MethodGet, "/api/v1/notes/"+url.PathEscape(id), nil, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// ListRecent returns the most recently created notes via GET
// /api/v1/notes?limit=N&offset=0 (the server sorts by created_at DESC).
func (c *Client) ListRecent(ctx context.Context, limit int) ([]Note, error) {
	var page struct {
		Notes []Note `json:"notes"`
	}
	path := fmt.Sprintf("/api/v1/notes?limit=%d&offset=0", limit)
	if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	return page.Notes, nil
}

// Search runs the API's hybrid full-text + vector search via GET
// /api/v1/search?q=... The server already ranks and caps results, so this
// only trims down to limit when the caller asked for fewer.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Note, error) {
	var page struct {
		Notes []Note `json:"notes"`
	}
	path := "/api/v1/search?q=" + url.QueryEscape(query)
	if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	if limit > 0 && len(page.Notes) > limit {
		page.Notes = page.Notes[:limit]
	}
	return page.Notes, nil
}
