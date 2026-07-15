# Mira

Application de prise de notes en ligne de commande, écrite en Go. Le dépôt suit une progression pédagogique : exercices d'échauffement, puis les TP qui construisent le projet fil rouge Mira.

## Structure

```
mira/
├── go-warmup/   # 5 exercices Go indépendants (bases du langage)
├── tp-1/        # Mira CLI locale (stockage JSONL sur disque)
├── tp-2/        # Mira API HTTP (stockage en mémoire, spec OpenAPI)
├── tp-3/        # Concurrence et goroutines (exercices indépendants)
└── tp-4/        # Mira v2 : PostgreSQL, enrichissement asynchrone, recherche hybride
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

### `tp-3/`

Exercices sur la concurrence en Go : goroutines, `sync.WaitGroup`, channels, worker pool, `select`/`time.After`, race condition (observée puis corrigée avec `sync.Mutex`), et un bonus sur `context.WithTimeout`. Chaque exercice est un `package main` indépendant (pas de `go.mod` séparé). Voir [tp-3/notes.md](tp-3/notes.md) et [tp-3/commands.md](tp-3/commands.md).

```bash
go build ./tp-3/...
go run ./tp-3/exo1
```

### `tp-4/`

Troisième évolution de Mira, sous forme d'un dossier séparé (`tp-1/`/`tp-2/` restent figés) : `tp-4/api` remplace le stockage en mémoire de tp-2 par **PostgreSQL** (pgx, migrations via golang-migrate, transaction sur création de note + tags), ajoute un **enrichissement automatique** asynchrone (tags/résumé/score/embedding simulés localement, calculés par un pool de workers borné avec timeout `context`, statut `enrichment_status` pending/done/failed) et une **recherche hybride** full-text + similarité vectorielle (pgvector, index GIN + HNSW). `tp-4/cli` fait évoluer tp-1 pour parler à cette API en HTTP au lieu du fichier JSONL local — condition nécessaire pour que toute note créée/modifiée déclenche l'enrichissement. Voir [tp-4/notes.md](tp-4/notes.md) et [tp-4/commands.md](tp-4/commands.md).

gitbash
```bash
docker compose -f tp-4/docker-compose.yml up -d
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api
go run ./tp-4/cli
```
