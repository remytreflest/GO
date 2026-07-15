package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func worker(ctx context.Context, id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		delai := time.Duration(0)
		if id == 1 {
			delai = 2 * time.Second
		}

		select {
		case <-time.After(delai):
		case <-ctx.Done():
			fmt.Printf("worker %d annulé (job %d abandonné)\n", id, j)
			return
		}

		select {
		case resultats <- j * j:
		case <-ctx.Done():
			fmt.Printf("worker %d annulé avant d'avoir pu envoyer le résultat du job %d\n", id, j)
			return
		}
	}
}

func main() {
	const nbJobs = 20
	const nbWorkers = 4

	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(ctx, w, jobs, resultats, &wg)
	}

	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(resultats)
	}()

boucle:
	for {
		select {
		case r, ok := <-resultats:
			if !ok {
				break boucle
			}
			fmt.Println("résultat :", r)
		case <-ctx.Done():
			fmt.Println("délai global d'1s dépassé, arrêt du traitement")
			break boucle
		}
	}
}
