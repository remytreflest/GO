package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	notes := []*Note{
		NewNote("Go basics", "Go is a statically typed, compiled language designed by Google."),
		NewNote("REST API", "REST stands for Representational State Transfer. It uses HTTP verbs."),
		NewNote("Go interfaces", "Interfaces in Go are implemented implicitly — no implements keyword needed."),
		NewNote("Docker", "Docker containers package code and dependencies together for portability."),
	}
	notes[0].AddTag("go")
	notes[0].AddTag("programming")
	notes[1].AddTag("api")
	notes[1].AddTag("rest")
	notes[2].AddTag("go")
	notes[2].AddTag("interfaces")
	notes[3].AddTag("devops")

	path := "notes.json"
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create:", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(notes); err != nil {
		f.Close()
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	f.Close()

	loaded, err := LoadFromFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d notes\n\n", len(loaded))
	fmt.Println(`Notes tagged "go":`)
	for _, n := range loaded {
		for _, t := range n.Tags {
			if t == "go" {
				fmt.Printf("  [%s] %s\n", n.Title, n.Preview())
				break
			}
		}
	}
}
