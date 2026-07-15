package main

import (
	"fmt"
	"time"
)

func afficherLettres() {
	lettres := []string{"a", "b", "c", "d", "e"}
	for _, l := range lettres {
		fmt.Println(l)
		time.Sleep(50 * time.Millisecond)
	}
}

func afficherChiffres() {
	for i := 1; i <= 5; i++ {
		fmt.Println(i)
	}
}

func main() {
	go afficherLettres()
	afficherChiffres()
	time.Sleep(100 * time.Millisecond)
}
