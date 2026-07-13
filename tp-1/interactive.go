package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mira/tp-1/internal/notes"
	"mira/tp-1/internal/search"
)

func runInteractive(store notes.Store) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== Mira — mode interactif ===")
	fmt.Println("Tapez 'quitter' pour sortir.")

	for {
		fmt.Print("\n> ajouter | lire | supprimer | lister | rechercher | quitter\n: ")
		if !scanner.Scan() {
			break
		}
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "ajouter":
			interactiveAdd(store, scanner)
		case "lire":
			interactiveGet(store, scanner)
		case "supprimer":
			interactiveDelete(store, scanner)
		case "lister":
			cmdList(store)
		case "rechercher":
			interactiveSearch(store, scanner)
		case "quitter", "q", "exit":
			fmt.Println("Au revoir.")
			return
		default:
			fmt.Println("Commande inconnue. Réessayez.")
		}
	}
}

func prompt(scanner *bufio.Scanner, label string) string {
	fmt.Print(label)
	scanner.Scan()
	return strings.TrimSpace(scanner.Text())
}

func interactiveAdd(store notes.Store, scanner *bufio.Scanner) {
	title := prompt(scanner, "  Titre    : ")
	if title == "" {
		fmt.Println("  Le titre est obligatoire.")
		return
	}
	content := prompt(scanner, "  Contenu  : ")
	tagsRaw := prompt(scanner, "  Tags (virgule) : ")

	n := notes.New(title, content)
	if tagsRaw != "" {
		for _, tag := range strings.Split(tagsRaw, ",") {
			n.AddTag(strings.TrimSpace(tag))
		}
	}

	if err := store.Save(n); err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	fmt.Printf("  Ajoutée [%s] %s\n", n.ID, n.Title)
}

func interactiveGet(store notes.Store, scanner *bufio.Scanner) {
	cmdList(store)
	id := prompt(scanner, "  ID : ")
	if id == "" {
		return
	}
	n, err := store.Get(id)
	if err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	fmt.Printf("\n  ID      : %s\n", n.ID)
	fmt.Printf("  Titre   : %s\n", n.Title)
	fmt.Printf("  Tags    : %s\n", strings.Join(n.Tags, ", "))
	fmt.Printf("  Contenu : %s\n", n.Content)
}

func interactiveDelete(store notes.Store, scanner *bufio.Scanner) {
	cmdList(store)
	id := prompt(scanner, "  ID à supprimer : ")
	if id == "" {
		return
	}
	if err := store.Delete(id); err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	fmt.Printf("  Note %s supprimée.\n", id)
}

func interactiveSearch(store notes.Store, scanner *bufio.Scanner) {
	query := prompt(scanner, "  Recherche : ")
	if query == "" {
		return
	}
	all, err := store.All()
	if err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	results := search.Filter(all, query)
	if len(results) == 0 {
		fmt.Printf("  Aucun résultat pour %q\n", query)
		return
	}
	fmt.Printf("  %d résultat(s) pour %q :\n", len(results), query)
	for _, n := range results {
		fmt.Printf("    [%s] %s — %s\n", n.ID, n.Title, n.Preview())
	}
}
