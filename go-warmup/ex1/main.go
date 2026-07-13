package main

import (
	"fmt"
	"os"
)

const MaxDisplay = 10

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no arguments provided")
		os.Exit(1)
	}

	fmt.Printf("Total: %d word(s)\n", len(args))

	displayed := 0
	for _, word := range args {
		if len(word) > 4 {
			if displayed < MaxDisplay {
				fmt.Println(" ", word)
			}
			displayed++
		}
	}
	fmt.Printf("Words longer than 4 chars: %d\n", displayed)
	if displayed > MaxDisplay {
		fmt.Printf("(showing first %d of %d)\n", MaxDisplay, displayed)
	}
}
