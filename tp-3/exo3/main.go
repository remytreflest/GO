package main

import "fmt"

func sommePartielle(nums []int, resultat chan<- int) {
	somme := 0
	for _, n := range nums {
		somme += n
	}
	resultat <- somme
}

func main() {
	const n = 1000
	nums := make([]int, n)
	for i := range nums {
		nums[i] = i + 1
	}

	const nbMorceaux = 4
	taille := n / nbMorceaux
	resultat := make(chan int)

	for i := 0; i < nbMorceaux; i++ {
		debut := i * taille
		fin := debut + taille
		go sommePartielle(nums[debut:fin], resultat)
	}

	somme := 0
	for i := 0; i < nbMorceaux; i++ {
		somme += <-resultat
	}

	attendu := n * (n + 1) / 2
	fmt.Println("Somme calculée :", somme)
	fmt.Println("Somme attendue :", attendu)
}
