package tools

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp-5/internal/miraclient"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeMira implements miraAPI with per-test function fields; leave a field
// nil in a test that never exercises that call.
type fakeMira struct {
	searchFn     func(ctx context.Context, query string, limit int) ([]miraclient.Note, error)
	getFn        func(ctx context.Context, id string) (*miraclient.Note, error)
	createFn     func(ctx context.Context, title, content string, tags []string) (*miraclient.Note, error)
	listRecentFn func(ctx context.Context, limit int) ([]miraclient.Note, error)
}

func (f *fakeMira) Search(ctx context.Context, query string, limit int) ([]miraclient.Note, error) {
	return f.searchFn(ctx, query, limit)
}

func (f *fakeMira) Get(ctx context.Context, id string) (*miraclient.Note, error) {
	return f.getFn(ctx, id)
}

func (f *fakeMira) Create(ctx context.Context, title, content string, tags []string) (*miraclient.Note, error) {
	return f.createFn(ctx, title, content, tags)
}

func (f *fakeMira) ListRecent(ctx context.Context, limit int) ([]miraclient.Note, error) {
	return f.listRecentFn(ctx, limit)
}

var fixedTime = time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

// --- SearchNotes ---

func TestSearchNotes_Success(t *testing.T) {
	var gotLimit int
	h := New(&fakeMira{searchFn: func(_ context.Context, query string, limit int) ([]miraclient.Note, error) {
		gotLimit = limit
		if query != "channels go" {
			t.Fatalf("unexpected query: %q", query)
		}
		return []miraclient.Note{{ID: "1", Title: "Go channels", Summary: "s", CreatedAt: fixedTime}}, nil
	}}, testLogger())

	_, out, err := h.SearchNotes(context.Background(), nil, SearchNotesInput{Query: "channels go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != defaultSearchLimit {
		t.Fatalf("expected default limit %d, got %d", defaultSearchLimit, gotLimit)
	}
	if out.Total != 1 || out.Notes[0].ID != "1" || out.Notes[0].CreatedAt != "2026-07-16T10:00:00Z" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestSearchNotes_LimitClampedToMax(t *testing.T) {
	var gotLimit int
	h := New(&fakeMira{searchFn: func(_ context.Context, _ string, limit int) ([]miraclient.Note, error) {
		gotLimit = limit
		return nil, nil
	}}, testLogger())

	if _, _, err := h.SearchNotes(context.Background(), nil, SearchNotesInput{Query: "go", Limit: 9999}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != maxSearchLimit {
		t.Fatalf("expected clamped limit %d, got %d", maxSearchLimit, gotLimit)
	}
}

func TestSearchNotes_EmptyQueryRejected(t *testing.T) {
	h := New(&fakeMira{}, testLogger())
	if _, _, err := h.SearchNotes(context.Background(), nil, SearchNotesInput{Query: "   "}); err == nil {
		t.Fatalf("expected an error for an empty query")
	}
}

func TestSearchNotes_UpstreamErrorWrapped(t *testing.T) {
	h := New(&fakeMira{searchFn: func(context.Context, string, int) ([]miraclient.Note, error) {
		return nil, errors.New("boom")
	}}, testLogger())

	_, _, err := h.SearchNotes(context.Background(), nil, SearchNotesInput{Query: "go"})
	if err == nil {
		t.Fatalf("expected an error")
	}
}

// --- GetNote ---

func TestGetNote_Success(t *testing.T) {
	h := New(&fakeMira{getFn: func(_ context.Context, id string) (*miraclient.Note, error) {
		if id != "abc" {
			t.Fatalf("unexpected id: %q", id)
		}
		return &miraclient.Note{ID: "abc", Title: "Go", Content: "full content", CreatedAt: fixedTime, UpdatedAt: fixedTime}, nil
	}}, testLogger())

	_, out, err := h.GetNote(context.Background(), nil, GetNoteInput{ID: "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "abc" || out.Content != "full content" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestGetNote_EmptyIDRejected(t *testing.T) {
	h := New(&fakeMira{}, testLogger())
	if _, _, err := h.GetNote(context.Background(), nil, GetNoteInput{ID: " "}); err == nil {
		t.Fatalf("expected an error for an empty id")
	}
}

func TestGetNote_NotFoundPassesThroughSentinel(t *testing.T) {
	h := New(&fakeMira{getFn: func(context.Context, string) (*miraclient.Note, error) {
		return nil, miraclient.ErrNotFound
	}}, testLogger())

	_, _, err := h.GetNote(context.Background(), nil, GetNoteInput{ID: "missing"})
	if !errors.Is(err, miraclient.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// --- AddNote ---

func TestAddNote_Success(t *testing.T) {
	h := New(&fakeMira{createFn: func(_ context.Context, title, content string, tags []string) (*miraclient.Note, error) {
		if title != "Go channels" || content != "notes about channels" || len(tags) != 2 {
			t.Fatalf("unexpected args: %q %q %v", title, content, tags)
		}
		return &miraclient.Note{ID: "new-id", Title: title, Content: content, Tags: tags, EnrichmentStatus: "pending", CreatedAt: fixedTime, UpdatedAt: fixedTime}, nil
	}}, testLogger())

	_, out, err := h.AddNote(context.Background(), nil, AddNoteInput{Title: "Go channels", Content: "notes about channels", Tags: []string{"go", "concurrency"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID != "new-id" || out.EnrichmentStatus != "pending" {
		t.Fatalf("unexpected output: %+v", out)
	}
}

func TestAddNote_EmptyTitleRejected(t *testing.T) {
	h := New(&fakeMira{}, testLogger())
	if _, _, err := h.AddNote(context.Background(), nil, AddNoteInput{Title: " ", Content: "x"}); err == nil {
		t.Fatalf("expected an error for an empty title")
	}
}

func TestAddNote_EmptyContentRejected(t *testing.T) {
	h := New(&fakeMira{}, testLogger())
	if _, _, err := h.AddNote(context.Background(), nil, AddNoteInput{Title: "x", Content: " "}); err == nil {
		t.Fatalf("expected an error for empty content")
	}
}

func TestAddNote_ValidationErrorPassesThroughSentinel(t *testing.T) {
	h := New(&fakeMira{createFn: func(context.Context, string, string, []string) (*miraclient.Note, error) {
		return nil, miraclient.ErrValidation
	}}, testLogger())

	_, _, err := h.AddNote(context.Background(), nil, AddNoteInput{Title: "x", Content: "y"})
	if !errors.Is(err, miraclient.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

// --- ListRecentNotes ---

func TestListRecentNotes_Success(t *testing.T) {
	var gotLimit int
	h := New(&fakeMira{listRecentFn: func(_ context.Context, limit int) ([]miraclient.Note, error) {
		gotLimit = limit
		return []miraclient.Note{{ID: "1", CreatedAt: fixedTime}, {ID: "2", CreatedAt: fixedTime}}, nil
	}}, testLogger())

	_, out, err := h.ListRecentNotes(context.Background(), nil, ListRecentNotesInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLimit != defaultRecentLimit || out.Total != 2 {
		t.Fatalf("unexpected output: limit=%d out=%+v", gotLimit, out)
	}
}

func TestListRecentNotes_UpstreamErrorWrapped(t *testing.T) {
	h := New(&fakeMira{listRecentFn: func(context.Context, int) ([]miraclient.Note, error) {
		return nil, errors.New("boom")
	}}, testLogger())

	if _, _, err := h.ListRecentNotes(context.Background(), nil, ListRecentNotesInput{Limit: -1}); err == nil {
		t.Fatalf("expected an error")
	}
}

// --- clamp ---

func TestClamp(t *testing.T) {
	cases := []struct {
		limit, def, max, want int
	}{
		{0, 10, 50, 10},
		{-5, 10, 50, 10},
		{5, 10, 50, 5},
		{1000, 10, 50, 50},
	}
	for _, c := range cases {
		if got := clamp(c.limit, c.def, c.max); got != c.want {
			t.Fatalf("clamp(%d,%d,%d) = %d, want %d", c.limit, c.def, c.max, got, c.want)
		}
	}
}

// --- recovered ---

func TestRecovered_ConvertsPanicToCleanError(t *testing.T) {
	h := New(&fakeMira{searchFn: func(context.Context, string, int) ([]miraclient.Note, error) {
		panic("boom")
	}}, testLogger())

	wrapped := recovered("search_notes", h.SearchNotes, h.Logger)
	_, _, err := wrapped(context.Background(), nil, SearchNotesInput{Query: "go"})
	if err == nil {
		t.Fatalf("expected the panic to be converted into an error")
	}
}

func TestRecovered_PassesThroughOnSuccess(t *testing.T) {
	h := New(&fakeMira{searchFn: func(context.Context, string, int) ([]miraclient.Note, error) {
		return []miraclient.Note{{ID: "1"}}, nil
	}}, testLogger())

	wrapped := recovered("search_notes", h.SearchNotes, h.Logger)
	_, out, err := wrapped(context.Background(), nil, SearchNotesInput{Query: "go"})
	if err != nil || out.Total != 1 {
		t.Fatalf("unexpected result: out=%+v err=%v", out, err)
	}
}

// --- Register: end-to-end over an in-memory MCP transport, exercising the
// same path Claude Code uses (tools/list, tools/call).

func TestRegister_ListsAllFourTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "mira-test"}, nil)
	New(&fakeMira{}, testLogger()).Register(server)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Wait()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	want := map[string]bool{"search_notes": false, "get_note": false, "add_note": false, "list_recent_notes": false}
	for _, tool := range res.Tools {
		if _, ok := want[tool.Name]; !ok {
			t.Fatalf("unexpected tool registered: %s", tool.Name)
		}
		if tool.Description == "" {
			t.Fatalf("tool %s has no description", tool.Name)
		}
		want[tool.Name] = true
	}
	for name, found := range want {
		if !found {
			t.Fatalf("expected tool %s to be registered", name)
		}
	}
}

func TestRegister_CallToolEndToEnd(t *testing.T) {
	fake := &fakeMira{
		createFn: func(_ context.Context, title, content string, tags []string) (*miraclient.Note, error) {
			return &miraclient.Note{ID: "new-id", Title: title, Content: content, EnrichmentStatus: "pending", CreatedAt: fixedTime, UpdatedAt: fixedTime}, nil
		},
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "mira-test"}, nil)
	New(fake, testLogger()).Register(server)

	ctx := context.Background()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect: %v", err)
	}
	defer serverSession.Wait()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	defer clientSession.Close()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_note",
		Arguments: map[string]any{"title": "Go channels", "content": "notes"},
	})
	if err != nil {
		t.Fatalf("CallTool: unexpected protocol error: %v", err)
	}
	if res.IsError {
		t.Fatalf("expected a successful tool result, got IsError with content %+v", res.Content)
	}

	// A bad call (missing required "title") must come back as a clean tool
	// error, never a protocol-level crash.
	res, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "add_note",
		Arguments: map[string]any{"content": "notes"},
	})
	if err != nil {
		t.Fatalf("CallTool: unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected a tool error for missing required field")
	}
}
