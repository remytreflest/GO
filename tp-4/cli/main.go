package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"mira/tp-4/cli/internal/apiclient"
	"mira/tp-4/cli/internal/notes"
)

const defaultAPIURL = "http://localhost:8080"

func main() {
	apiURL := os.Getenv("MIRA_API_URL")
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	client := apiclient.New(apiURL)

	if err := client.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "cannot reach Mira API at %s: %v\n", apiURL, err)
		fmt.Fprintln(os.Stderr, "hint: is the API running? see tp-4/api/commands.md")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		runInteractive(client)
		return
	}

	switch os.Args[1] {
	case "add":
		cmdAdd(client)
	case "list":
		cmdList(client)
	case "search":
		cmdSearch(client)
	case "get":
		cmdGet(client)
	case "update":
		cmdUpdate(client)
	case "delete":
		cmdDelete(client)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func cmdAdd(store notes.Store) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	tags := fs.String("tags", "", "comma-separated tags")
	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 2 {
		// flag.Parse stops at the first non-flag argument, so flags must
		// come before the positional title/content, not after.
		fmt.Fprintln(os.Stderr, `usage: mira add [-tags tag1,tag2] "titre" "contenu"`)
		os.Exit(1)
	}

	n := notes.New(args[0], args[1])
	if *tags != "" {
		for _, tag := range strings.Split(*tags, ",") {
			n.AddTag(strings.TrimSpace(tag))
		}
	}

	if err := store.Save(n); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("Added [%s] %s (enrichment: %s)\n", n.ID, n.Title, n.EnrichmentStatus)
}

func cmdList(store notes.Store) {
	list, err := store.List()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(list) == 0 {
		fmt.Println("No notes.")
		return
	}
	fmt.Printf("%d dernière(s) note(s) :\n", len(list))
	for _, n := range list {
		printNoteSummary(n)
	}
}

func cmdSearch(client *apiclient.Client) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mira search <query>")
		os.Exit(1)
	}
	query := strings.Join(os.Args[2:], " ")
	results, err := client.Search(query)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Printf("No results for %q\n", query)
		return
	}
	fmt.Printf("%d result(s) for %q:\n", len(results), query)
	for _, n := range results {
		printNoteSummary(n)
	}
}

func cmdGet(store notes.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mira get <id>")
		os.Exit(1)
	}
	n, err := store.Get(os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printNoteDetail(n)
}

func cmdUpdate(client *apiclient.Client) {
	fs := flag.NewFlagSet("update", flag.ExitOnError)
	title := fs.String("title", "", "new title")
	content := fs.String("content", "", "new content")
	tags := fs.String("tags", "", "comma-separated tags (replaces existing tags)")
	fs.Parse(os.Args[2:])

	args := fs.Args()
	if len(args) < 1 {
		// flag.Parse stops at the first non-flag argument, so flags must
		// come before the positional id, not after.
		fmt.Fprintln(os.Stderr, `usage: mira update [-title "..."] [-content "..."] [-tags tag1,tag2] <id>`)
		os.Exit(1)
	}
	id := args[0]

	var titlePtr, contentPtr *string
	if *title != "" {
		titlePtr = title
	}
	if *content != "" {
		contentPtr = content
	}
	var tagList []string
	if *tags != "" {
		tagList = strings.Split(*tags, ",")
		for i := range tagList {
			tagList[i] = strings.TrimSpace(tagList[i])
		}
	}
	if titlePtr == nil && contentPtr == nil && tagList == nil {
		fmt.Fprintln(os.Stderr, "error: at least one of -title, -content, -tags must be provided")
		os.Exit(1)
	}

	n, err := client.Update(id, titlePtr, contentPtr, tagList)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("Updated [%s] %s (enrichment: %s)\n", n.ID, n.Title, n.EnrichmentStatus)
}

func cmdDelete(store notes.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mira delete <id>")
		os.Exit(1)
	}
	id := os.Args[2]
	if err := store.Delete(id); err != nil {
		if errors.Is(err, notes.ErrNotFound) {
			fmt.Fprintf(os.Stderr, "note %s not found\n", id)
		} else {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
		os.Exit(1)
	}
	fmt.Printf("Deleted %s\n", id)
}

func printNoteSummary(n *notes.Note) {
	fmt.Printf("  [%s] %s (%s)\n", n.ID, n.Title, n.EnrichmentStatus)
	if n.Content != "" {
		fmt.Printf("          %s\n", n.Preview())
	}
}

func printNoteDetail(n *notes.Note) {
	fmt.Printf("ID:                %s\n", n.ID)
	fmt.Printf("Title:             %s\n", n.Title)
	fmt.Printf("Tags:              %s\n", strings.Join(n.Tags, ", "))
	fmt.Printf("Content:           %s\n", n.Content)
	fmt.Printf("Enrichment status: %s\n", n.EnrichmentStatus)
	if n.EnrichmentStatus == "done" {
		fmt.Printf("Summary:           %s\n", n.Summary)
		fmt.Printf("Score:             %.2f\n", n.Score)
	}
}

func printUsage() {
	// Flags are listed before positional arguments throughout: Go's flag
	// package stops parsing at the first non-flag argument, so e.g.
	// `mira add "titre" "contenu" -tags x` would silently ignore -tags.
	fmt.Print(`Usage:
  mira add [-tags tag1,tag2] "titre" "contenu"
  mira list
  mira search <query>
  mira get <id>
  mira update [-title "..."] [-content "..."] [-tags tag1,tag2] <id>
  mira delete <id>

Reads MIRA_API_URL from the environment (default http://localhost:8080).
`)
}
