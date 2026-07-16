package main

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestResolveAPIURL_Default(t *testing.T) {
	t.Setenv("MIRA_API_URL", "")
	if got := resolveAPIURL(); got != defaultAPIURL {
		t.Fatalf("expected default %q, got %q", defaultAPIURL, got)
	}
}

func TestResolveAPIURL_FromEnv(t *testing.T) {
	t.Setenv("MIRA_API_URL", "http://example.com:9000")
	if got := resolveAPIURL(); got != "http://example.com:9000" {
		t.Fatalf("expected env override, got %q", got)
	}
}

func TestBuildServer_RegistersTools(t *testing.T) {
	server := buildServer(newLogger(), "http://localhost:8080")
	if server == nil {
		t.Fatal("expected a non-nil server")
	}

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
	if len(res.Tools) != 4 {
		t.Fatalf("expected 4 registered tools, got %d", len(res.Tools))
	}
}
