package main

import (
	"fmt"
	"sync"
)

func main() {
	compteur := 0
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			compteur++
		}()
	}

	wg.Wait()
	fmt.Println("Compteur final :", compteur)
}
