package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"mira/tp-1/internal/notes"
	"mira/tp-1/internal/search"
)

func main() {
	store, err := notes.NewJSONLStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "init:", err)
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		runInteractive(store)
		return
	}

	switch os.Args[1] {
	case "add":
		cmdAdd(store)
	case "list":
		cmdList(store)
	case "search":
		cmdSearch(store)
	case "get":
		cmdGet(store)
	case "delete":
		cmdDelete(store)
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
		fmt.Fprintln(os.Stderr, `usage: mira add "titre" "contenu" [-tags tag1,tag2]`)
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
	fmt.Printf("Added [%s] %s\n", n.ID, n.Title)
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
		fmt.Printf("  [%s] %s\n", n.ID, n.Title)
		if n.Content != "" {
			fmt.Printf("          %s\n", n.Preview())
		}
	}
}

func cmdSearch(store notes.Store) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: mira search <query>")
		os.Exit(1)
	}
	query := strings.Join(os.Args[2:], " ")
	all, err := store.All()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	results := search.Filter(all, query)
	if len(results) == 0 {
		fmt.Printf("No results for %q\n", query)
		return
	}
	fmt.Printf("%d result(s) for %q:\n", len(results), query)
	for _, n := range results {
		fmt.Printf("  [%s] %s — %s\n", n.ID, n.Title, n.Preview())
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
	fmt.Printf("ID:      %s\n", n.ID)
	fmt.Printf("Title:   %s\n", n.Title)
	fmt.Printf("Tags:    %s\n", strings.Join(n.Tags, ", "))
	fmt.Printf("Content: %s\n", n.Content)
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

func printUsage() {
	fmt.Print(`Usage:
  mira add "titre" "contenu" [-tags tag1,tag2]
  mira list
  mira search <query>
  mira get <id>
  mira delete <id>
`)
}
