package main

import (
	"fmt"
	"sync"
)

func worker(id int, jobs <-chan int, resultats chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	for j := range jobs {
		resultats <- j * j
	}
}

func main() {
	const nbJobs = 20
	const nbWorkers = 4

	jobs := make(chan int, nbJobs)
	resultats := make(chan int, nbJobs)
	var wg sync.WaitGroup

	for w := 1; w <= nbWorkers; w++ {
		wg.Add(1)
		go worker(w, jobs, resultats, &wg)
	}

	for j := 1; j <= nbJobs; j++ {
		jobs <- j
	}
	close(jobs)

	wg.Wait()
	close(resultats)

	// L'ordre n'est pas garanti : les 4 workers tournent en parallèle et
	// piochent dans `jobs` dès qu'ils sont libres. Le scheduler Go décide de
	// l'ordre d'exécution des goroutines, qui dépend du timing de chacune
	// (durée du calcul, moment où le worker se libère, etc.), pas de l'ordre
	// d'envoi des jobs.
	for r := range resultats {
		fmt.Println(r)
	}
}
