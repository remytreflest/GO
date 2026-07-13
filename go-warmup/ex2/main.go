package main

import (
	"fmt"
	"os"
	"sort"
)

type tagCount struct {
	tag   string
	count int
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: no tags provided")
		os.Exit(1)
	}

	counts := make(map[string]int)
	for _, tag := range args {
		counts[tag]++
	}

	pairs := make([]tagCount, 0, len(counts))
	for k, v := range counts {
		pairs = append(pairs, tagCount{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].tag < pairs[j].tag
	})

	fmt.Println("All tags (by frequency):")
	for _, p := range pairs {
		fmt.Printf("  %-15s %d\n", p.tag, p.count)
	}

	fmt.Println("\nTags with more than one occurrence:")
	found := false
	for _, p := range pairs {
		if p.count > 1 {
			fmt.Printf("  %-15s %d\n", p.tag, p.count)
			found = true
		}
	}
	if !found {
		fmt.Println("  (none)")
	}
}
