// Package tools implements the MCP tools exposed by cmd/mira-mcp:
// search_notes, get_note, add_note and list_recent_notes. Every handler
// calls the mira HTTP API through miraclient (never a database directly),
// which is what guarantees that a note added here goes through the same
// asynchronous enrichment pipeline as the CLI.
package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp-5/internal/miraclient"
)

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 50
	defaultRecentLimit = 10
	maxRecentLimit     = 100
)

// miraAPI is the subset of *miraclient.Client used by tool handlers, so
// tests can inject a fake instead of spinning up an HTTP server.
type miraAPI interface {
	Search(ctx context.Context, query string, limit int) ([]miraclient.Note, error)
	Get(ctx context.Context, id string) (*miraclient.Note, error)
	Create(ctx context.Context, title, content string, tags []string) (*miraclient.Note, error)
	ListRecent(ctx context.Context, limit int) ([]miraclient.Note, error)
}

var _ miraAPI = (*miraclient.Client)(nil)

// Handlers wires MCP tool handlers to a mira API client.
type Handlers struct {
	Client miraAPI
	Logger *slog.Logger
}

// New builds a Handlers. logger must not be nil; use slog.Default() when the
// caller has no specific preference.
func New(client miraAPI, logger *slog.Logger) *Handlers {
	return &Handlers{Client: client, Logger: logger}
}

// Register adds all four mira tools to server, wrapping each handler so a
// panic is turned into a clean tool error (IsError) instead of crashing the
// stdio server for every other in-flight call.
func (h *Handlers) Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "search_notes",
		Description: "Recherche des notes mira par mots-clés ou question en langage naturel, en combinant " +
			"recherche plein texte et similarité vectorielle (recherche hybride). Retourne un aperçu de " +
			"chaque note (titre, résumé, tags, statut d'enrichissement) — utilisez get_note avec l'id " +
			"retourné pour lire le contenu complet. À utiliser pour retrouver une note existante, par " +
			"exemple avant d'en créer une nouvelle sur le même sujet.",
	}, recovered("search_notes", h.SearchNotes, h.Logger))

	mcp.AddTool(server, &mcp.Tool{
		Name: "get_note",
		Description: "Récupère une note mira complète (titre, contenu intégral, tags, résumé et statut " +
			"d'enrichissement) à partir de son identifiant. À utiliser une fois l'identifiant connu, " +
			"typiquement après un appel à search_notes ou list_recent_notes.",
	}, recovered("get_note", h.GetNote, h.Logger))

	mcp.AddTool(server, &mcp.Tool{
		Name: "add_note",
		Description: "Crée une nouvelle note mira avec un titre, un contenu et des tags optionnels. La " +
			"note est immédiatement persistée puis enrichie de façon asynchrone côté serveur (tags " +
			"suggérés, résumé, score) : son enrichment_status vaut \"pending\" juste après la création " +
			"puis passe à \"done\" quelques instants plus tard.",
	}, recovered("add_note", h.AddNote, h.Logger))

	mcp.AddTool(server, &mcp.Tool{
		Name: "list_recent_notes",
		Description: "Liste les notes mira les plus récemment créées, triées de la plus récente à la plus " +
			"ancienne, avec un aperçu de chacune (titre, résumé, tags, statut d'enrichissement). À " +
			"utiliser pour retrouver rapidement une note qui vient d'être ajoutée sans connaître son " +
			"identifiant ni de mots-clés précis à rechercher.",
	}, recovered("list_recent_notes", h.ListRecentNotes, h.Logger))
}

// NoteSummary is a compact preview of a note, returned by search_notes and
// list_recent_notes. Fetch the full note with get_note when the content is
// needed.
type NoteSummary struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Summary          string   `json:"summary,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
}

// NoteDetail is the full representation of a note, returned by get_note and
// add_note.
type NoteDetail struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	Tags             []string `json:"tags,omitempty"`
	Summary          string   `json:"summary,omitempty"`
	Score            float64  `json:"score"`
	EnrichmentStatus string   `json:"enrichment_status"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

// SearchNotesInput is the input for the search_notes tool.
type SearchNotesInput struct {
	Query string `json:"query" jsonschema:"terme ou question en langage naturel à rechercher parmi les notes"`
	Limit int    `json:"limit,omitempty" jsonschema:"nombre maximum de résultats à retourner (défaut 10, max 50)"`
}

// SearchNotesOutput is the output of the search_notes tool.
type SearchNotesOutput struct {
	Notes []NoteSummary `json:"notes"`
	Total int           `json:"total"`
}

// SearchNotes implements the search_notes tool.
func (h *Handlers) SearchNotes(ctx context.Context, _ *mcp.CallToolRequest, in SearchNotesInput) (*mcp.CallToolResult, SearchNotesOutput, error) {
	query := strings.TrimSpace(in.Query)
	if query == "" {
		return nil, SearchNotesOutput{}, errors.New("query is required and cannot be empty")
	}

	notes, err := h.Client.Search(ctx, query, clamp(in.Limit, defaultSearchLimit, maxSearchLimit))
	if err != nil {
		return nil, SearchNotesOutput{}, friendlyErr("search notes", err)
	}
	return nil, SearchNotesOutput{Notes: toSummaries(notes), Total: len(notes)}, nil
}

// GetNoteInput is the input for the get_note tool.
type GetNoteInput struct {
	ID string `json:"id" jsonschema:"identifiant unique de la note à récupérer"`
}

// GetNote implements the get_note tool.
func (h *Handlers) GetNote(ctx context.Context, _ *mcp.CallToolRequest, in GetNoteInput) (*mcp.CallToolResult, NoteDetail, error) {
	id := strings.TrimSpace(in.ID)
	if id == "" {
		return nil, NoteDetail{}, errors.New("id is required and cannot be empty")
	}

	n, err := h.Client.Get(ctx, id)
	if err != nil {
		return nil, NoteDetail{}, friendlyErr(fmt.Sprintf("get note %q", id), err)
	}
	return nil, toDetail(*n), nil
}

// AddNoteInput is the input for the add_note tool.
type AddNoteInput struct {
	Title   string   `json:"title" jsonschema:"titre court et descriptif de la note"`
	Content string   `json:"content" jsonschema:"contenu complet de la note"`
	Tags    []string `json:"tags,omitempty" jsonschema:"tags optionnels à associer à la note"`
}

// AddNote implements the add_note tool.
func (h *Handlers) AddNote(ctx context.Context, _ *mcp.CallToolRequest, in AddNoteInput) (*mcp.CallToolResult, NoteDetail, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, NoteDetail{}, errors.New("title is required and cannot be empty")
	}
	if strings.TrimSpace(in.Content) == "" {
		return nil, NoteDetail{}, errors.New("content is required and cannot be empty")
	}

	n, err := h.Client.Create(ctx, title, in.Content, in.Tags)
	if err != nil {
		return nil, NoteDetail{}, friendlyErr("create note", err)
	}
	return nil, toDetail(*n), nil
}

// ListRecentNotesInput is the input for the list_recent_notes tool.
type ListRecentNotesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"nombre maximum de notes à retourner (défaut 10, max 100)"`
}

// ListRecentNotesOutput is the output of the list_recent_notes tool.
type ListRecentNotesOutput struct {
	Notes []NoteSummary `json:"notes"`
	Total int           `json:"total"`
}

// ListRecentNotes implements the list_recent_notes tool.
func (h *Handlers) ListRecentNotes(ctx context.Context, _ *mcp.CallToolRequest, in ListRecentNotesInput) (*mcp.CallToolResult, ListRecentNotesOutput, error) {
	notes, err := h.Client.ListRecent(ctx, clamp(in.Limit, defaultRecentLimit, maxRecentLimit))
	if err != nil {
		return nil, ListRecentNotesOutput{}, friendlyErr("list recent notes", err)
	}
	return nil, ListRecentNotesOutput{Notes: toSummaries(notes), Total: len(notes)}, nil
}

// clamp applies def when limit is not positive, then caps it at max.
func clamp(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

// friendlyErr turns a miraclient error into a clean, user-facing message:
// known sentinels pass through as-is, anything else is wrapped with the
// attempted action for context. miraclient never produces raw stack traces,
// so this never leaks internal details.
func friendlyErr(action string, err error) error {
	switch {
	case errors.Is(err, miraclient.ErrNotFound):
		return miraclient.ErrNotFound
	case errors.Is(err, miraclient.ErrValidation):
		return err
	default:
		return fmt.Errorf("failed to %s: %w", action, err)
	}
}

func toSummaries(notes []miraclient.Note) []NoteSummary {
	out := make([]NoteSummary, len(notes))
	for i, n := range notes {
		out[i] = NoteSummary{
			ID:               n.ID,
			Title:            n.Title,
			Summary:          n.Summary,
			Tags:             n.Tags,
			EnrichmentStatus: n.EnrichmentStatus,
			CreatedAt:        n.CreatedAt.Format(time.RFC3339),
		}
	}
	return out
}

func toDetail(n miraclient.Note) NoteDetail {
	return NoteDetail{
		ID:               n.ID,
		Title:            n.Title,
		Content:          n.Content,
		Tags:             n.Tags,
		Summary:          n.Summary,
		Score:            n.Score,
		EnrichmentStatus: n.EnrichmentStatus,
		CreatedAt:        n.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        n.UpdatedAt.Format(time.RFC3339),
	}
}

// recovered wraps a ToolHandlerFor so a panic inside it is logged and
// converted into a clean tool error, instead of crashing the whole stdio
// server process (which would take down every other in-flight tool call).
func recovered[In, Out any](name string, fn mcp.ToolHandlerFor[In, Out], logger *slog.Logger) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (res *mcp.CallToolResult, out Out, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("tool handler panicked", "tool", name, "panic", r)
				err = fmt.Errorf("internal error while running %s", name)
			}
		}()
		return fn(ctx, req, in)
	}
}
