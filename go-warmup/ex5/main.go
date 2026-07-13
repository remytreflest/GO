package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	var store NoteStore = NewMemoryStore()

	n1 := NewNote("1", "Go basics", "Go is a statically typed, compiled language.")
	n1.AddTag("go")
	n2 := NewNote("2", "REST API", "REST uses HTTP verbs: GET, POST, PUT, DELETE.")
	n2.AddTag("api")

	for _, n := range []*Note{n1, n2} {
		if err := store.Save(n); err != nil {
			fmt.Fprintln(os.Stderr, "save:", err)
			os.Exit(1)
		}
	}

	if err := store.Save(n1); !errors.Is(err, ErrDuplicate) {
		fmt.Fprintln(os.Stderr, "expected ErrDuplicate, got:", err)
		os.Exit(1)
	}
	fmt.Println("OK: duplicate rejected")

	blank := &Note{ID: "3"}
	if err := store.Save(blank); !errors.Is(err, ErrValidation) {
		fmt.Fprintln(os.Stderr, "expected ErrValidation, got:", err)
		os.Exit(1)
	}
	fmt.Println("OK: empty title rejected")

	n, err := store.Get("1")
	if err != nil {
		fmt.Fprintln(os.Stderr, "get:", err)
		os.Exit(1)
	}
	fmt.Printf("Found: [%s] %s — %s\n", n.ID, n.Title, n.Preview())

	if _, err := store.Get("unknown"); !errors.Is(err, ErrNotFound) {
		fmt.Fprintln(os.Stderr, "expected ErrNotFound, got:", err)
		os.Exit(1)
	}
	fmt.Println("OK: not found handled")

	fmt.Printf("\nAll notes (%d):\n", len(store.All()))
	for _, n := range store.All() {
		fmt.Printf("  [%s] %s\n", n.ID, n.Title)
	}
}
