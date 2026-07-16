// Command mira-mcp is an MCP server exposing mira's notes to AI agents
// (Claude Code, Claude Desktop, any MCP-aware IDE) over the stdio
// transport, so an agent can search, read and create notes during a
// conversation. Every tool call goes through mira's HTTP API — never the
// database directly — which is what triggers the API's asynchronous
// enrichment pipeline, exactly like tp-4/cli.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"mira/tp-5/internal/miraclient"
	"mira/tp-5/internal/tools"
)

// defaultAPIURL is used when MIRA_API_URL is unset.
const defaultAPIURL = "http://localhost:8080"

// version is the server's reported implementation version.
const version = "0.1.0"

func main() {
	logger := newLogger()
	apiURL := resolveAPIURL()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := buildServer(logger, apiURL)

	logger.Info("starting mira MCP server", "transport", "stdio", "mira_api_url", apiURL)
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		logger.Error("mira MCP server stopped with an error", "error", err)
		os.Exit(1)
	}
}

// newLogger builds a structured logger writing to stderr. In stdio
// transport, stdout is reserved exclusively for JSON-RPC protocol frames:
// anything else written there would corrupt the stream, so no log line may
// ever go to stdout.
func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// resolveAPIURL returns the mira API base URL, configurable via the
// MIRA_API_URL environment variable, defaulting to defaultAPIURL.
func resolveAPIURL() string {
	if v := os.Getenv("MIRA_API_URL"); v != "" {
		return v
	}
	return defaultAPIURL
}

// buildServer wires a mira API client and the four note tools into an MCP
// server, ready to be run over any transport.
func buildServer(logger *slog.Logger, apiURL string) *mcp.Server {
	client := miraclient.New(apiURL, &http.Client{})
	handlers := tools.New(client, logger)

	server := mcp.NewServer(&mcp.Implementation{Name: "mira", Version: version}, &mcp.ServerOptions{
		Logger: logger,
		Instructions: "Expose les notes de mira (recherche hybride, lecture, création) à un agent IA, " +
			"en passant systématiquement par l'API HTTP de mira pour que toute note créée déclenche " +
			"l'enrichissement automatique (tags, résumé, score, embedding).",
	})
	handlers.Register(server)
	return server
}
