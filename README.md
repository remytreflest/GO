# Mira

Application de prise de notes en ligne de commande, écrite en Go. Le dépôt suit une progression pédagogique : exercices d'échauffement, puis deux TP qui construisent le projet fil rouge Mira.

## Structure

```
mira/
├── go-warmup/   # 5 exercices Go indépendants (bases du langage)
├── tp-1/        # Mira CLI locale (stockage JSONL sur disque)
└── tp-2/        # Mira API HTTP (stockage en mémoire, spec OpenAPI)
```

### `go-warmup/`

Cinq mini-exercices indépendants pour prendre en main le langage, chacun dans son propre dossier (`ex1` à `ex5`) :

| Exercice | Sujet |
|---|---|
| `ex1` | Variables, types, zero values, `os.Args`, boucles |
| `ex2` | Slices, maps, tri (`sort.Slice`) |
| `ex3` | Structs, méthodes, receivers, `defer` |
| `ex4` | Interfaces implicites, erreurs typées, `sync.Mutex` |
| `ex5` | Tests unitaires, table-driven tests |

Chaque exercice contient `notes.md` (notions clés apprises) et `commands.md` (commandes pour l'exécuter/tester).

### `tp-1/`

Première version de Mira : une CLI qui persiste les notes dans un fichier JSON Lines (`~/.mira/notes.jsonl`), avec recherche et pattern repository (`notes.Store` / `JSONLStore`). Voir `tp-1/notes.md` et `tp-1/commands.md`.

### `tp-2/`

Deuxième version de Mira : une API HTTP (`/api/v1/notes`) construite sur `net/http` (Go 1.22+, routage par méthode + pattern), avec stockage en mémoire thread-safe, validation, middlewares (request ID, logging structuré, recovery, timeout) et spec OpenAPI générée via `swaggo/swag`. Voir [tp-2/README.md](tp-2/README.md) pour le détail des routes et des exemples `curl`.

```bash
go run ./tp-2/cmd/api
```
