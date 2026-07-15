package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mira/tp-4/cli/internal/apiclient"
	"mira/tp-4/cli/internal/notes"
)

func runInteractive(client *apiclient.Client) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== Mira — mode interactif (API) ===")
	fmt.Println("Tapez 'quitter' pour sortir.")

	for {
		fmt.Print("\n> ajouter | lire | modifier | supprimer | lister | rechercher | quitter\n: ")
		if !scanner.Scan() {
			break
		}
		switch strings.TrimSpace(strings.ToLower(scanner.Text())) {
		case "ajouter":
			interactiveAdd(client, scanner)
		case "lire":
			interactiveGet(client, scanner)
		case "modifier":
			interactiveUpdate(client, scanner)
		case "supprimer":
			interactiveDelete(client, scanner)
		case "lister":
			cmdList(client)
		case "rechercher":
			interactiveSearch(client, scanner)
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
	fmt.Printf("  Ajoutée [%s] %s (enrichissement : %s)\n", n.ID, n.Title, n.EnrichmentStatus)
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
	fmt.Printf("  Enrichissement : %s\n", n.EnrichmentStatus)
	if n.EnrichmentStatus == "done" {
		fmt.Printf("  Résumé  : %s\n", n.Summary)
		fmt.Printf("  Score   : %.2f\n", n.Score)
	}
}

func interactiveUpdate(client *apiclient.Client, scanner *bufio.Scanner) {
	cmdList(client)
	id := prompt(scanner, "  ID à modifier : ")
	if id == "" {
		return
	}
	title := prompt(scanner, "  Nouveau titre (vide = inchangé) : ")
	content := prompt(scanner, "  Nouveau contenu (vide = inchangé) : ")
	tagsRaw := prompt(scanner, "  Nouveaux tags, virgule (vide = inchangé) : ")

	var titlePtr, contentPtr *string
	if title != "" {
		titlePtr = &title
	}
	if content != "" {
		contentPtr = &content
	}
	var tags []string
	if tagsRaw != "" {
		for _, tag := range strings.Split(tagsRaw, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}
	if titlePtr == nil && contentPtr == nil && tags == nil {
		fmt.Println("  Rien à modifier.")
		return
	}

	n, err := client.Update(id, titlePtr, contentPtr, tags)
	if err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	fmt.Printf("  Modifiée [%s] %s (enrichissement : %s)\n", n.ID, n.Title, n.EnrichmentStatus)
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

func interactiveSearch(client *apiclient.Client, scanner *bufio.Scanner) {
	query := prompt(scanner, "  Recherche : ")
	if query == "" {
		return
	}
	results, err := client.Search(query)
	if err != nil {
		fmt.Println("  Erreur :", err)
		return
	}
	if len(results) == 0 {
		fmt.Printf("  Aucun résultat pour %q\n", query)
		return
	}
	fmt.Printf("  %d résultat(s) pour %q :\n", len(results), query)
	for _, n := range results {
		fmt.Printf("    [%s] %s — %s\n", n.ID, n.Title, n.Preview())
	}
}
