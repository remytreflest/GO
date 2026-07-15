package apiclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mira/tp-4/cli/internal/notes"
)

const (
	defaultTimeout = 10 * time.Second
	pageSize       = 100
	maxPages       = 1000 // safety cap against a misreported/looping total
)

// Client implements notes.Store over HTTP against the tp-4/api server.
type Client struct {
	baseURL string
	http    *http.Client
}

var _ notes.Store = (*Client)(nil)

func New(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// apiNote mirrors the server's note JSON (a superset of notes.Note: it also
// carries the enrichment fields), used to decode responses before copying
// values back into a *notes.Note.
type apiNote struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Tags             []string  `json:"tags"`
	Summary          string    `json:"summary"`
	Score            float64   `json:"score"`
	EnrichmentStatus string    `json:"enrichment_status"`
	CreatedAt        time.Time `json:"created_at"`
}

func (a apiNote) into(n *notes.Note) {
	n.ID = a.ID
	n.Title = a.Title
	n.Content = a.Content
	n.Tags = a.Tags
	n.Summary = a.Summary
	n.Score = a.Score
	n.EnrichmentStatus = a.EnrichmentStatus
	n.CreatedAt = a.CreatedAt
}

func toNotes(in []apiNote) []*notes.Note {
	out := make([]*notes.Note, len(in))
	for i, a := range in {
		n := &notes.Note{}
		a.into(n)
		out[i] = n
	}
	return out
}

// do sends an HTTP request, decodes the {data,error,request_id} envelope,
// and unmarshals Data into out (if non-nil) on success.
func (c *Client) do(method, path string, body, out any) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("call %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("decode response from %s: %w", c.baseURL, err)
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

// Ping verifies the API is reachable, so callers can fail fast with a clear
// message instead of a confusing timeout on the first real command.
func (c *Client) Ping() error {
	return c.do(http.MethodGet, "/api/v1/notes?limit=1", nil, nil)
}

// Save creates the note through POST /api/v1/notes and mutates n in place
// with the server-assigned ID/CreatedAt (the server owns ID generation, so
// the client-side placeholder from notes.New is always overwritten).
func (c *Client) Save(n *notes.Note) error {
	payload := struct {
		Title   string   `json:"title"`
		Content string   `json:"content"`
		Tags    []string `json:"tags,omitempty"`
	}{Title: n.Title, Content: n.Content, Tags: n.Tags}

	var created apiNote
	if err := c.do(http.MethodPost, "/api/v1/notes", payload, &created); err != nil {
		return err
	}
	created.into(n)
	return nil
}

func (c *Client) Get(id string) (*notes.Note, error) {
	var got apiNote
	if err := c.do(http.MethodGet, "/api/v1/notes/"+url.PathEscape(id), nil, &got); err != nil {
		return nil, err
	}
	n := &notes.Note{}
	got.into(n)
	return n, nil
}

func (c *Client) Delete(id string) error {
	return c.do(http.MethodDelete, "/api/v1/notes/"+url.PathEscape(id), nil, nil)
}

// List returns the 10 most recent notes, matching tp-1's original contract
// (the server already sorts by created_at DESC).
func (c *Client) List() ([]*notes.Note, error) {
	var page struct {
		Notes []apiNote `json:"notes"`
	}
	if err := c.do(http.MethodGet, "/api/v1/notes?limit=10&offset=0", nil, &page); err != nil {
		return nil, err
	}
	return toNotes(page.Notes), nil
}

// All paginates through every note. It exists to satisfy notes.Store; the
// CLI's search command bypasses it in favor of calling Search directly,
// since ranking only makes sense server-side (that's where the
// enrichment-produced embeddings live).
func (c *Client) All() ([]*notes.Note, error) {
	var all []*notes.Note
	offset := 0
	for i := 0; i < maxPages; i++ {
		var page struct {
			Notes []apiNote `json:"notes"`
			Total int       `json:"total"`
		}
		path := fmt.Sprintf("/api/v1/notes?limit=%d&offset=%d", pageSize, offset)
		if err := c.do(http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		if len(page.Notes) == 0 {
			break
		}
		all = append(all, toNotes(page.Notes)...)
		offset += len(page.Notes)
		if offset >= page.Total {
			break
		}
	}
	return all, nil
}

// Search calls the API's hybrid full-text + vector search.
func (c *Client) Search(query string) ([]*notes.Note, error) {
	var page struct {
		Notes []apiNote `json:"notes"`
	}
	path := "/api/v1/search?q=" + url.QueryEscape(query)
	if err := c.do(http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	return toNotes(page.Notes), nil
}

// Update calls PATCH /api/v1/notes/{id}. It is not part of notes.Store
// (tp-1 had no update command); tp-4/cli adds a new `update` subcommand
// purely to exercise the PATCH -> re-enrichment path end-to-end.
func (c *Client) Update(id string, title, content *string, tags []string) (*notes.Note, error) {
	payload := struct {
		Title   *string   `json:"title,omitempty"`
		Content *string   `json:"content,omitempty"`
		Tags    *[]string `json:"tags,omitempty"`
	}{Title: title, Content: content}
	if tags != nil {
		payload.Tags = &tags
	}

	var updated apiNote
	if err := c.do(http.MethodPatch, "/api/v1/notes/"+url.PathEscape(id), payload, &updated); err != nil {
		return nil, err
	}
	n := &notes.Note{}
	updated.into(n)
	return n, nil
}
