# Mira

Application de prise de notes en ligne de commande, écrite en Go. Le dépôt suit une progression
pédagogique : des exercices d'échauffement, puis le projet fil rouge Mira construit par les TP.
**Le projet actif est `tp-4` (API) + `tp-5` (serveur MCP)**, qui forment un seul système — le reste
(`go-warmup`, `tp-1`, `tp-2`, `tp-3`) ce sont des exercices d'apprentissage indépendants, figés.

## Structure

```
mira/
├── go-warmup/   # exercice : bases du langage Go
├── tp-1/        # exercice : Mira CLI locale (JSONL)
├── tp-2/        # exercice : Mira API HTTP en mémoire
├── tp-3/        # exercice : concurrence (goroutines, channels, worker pool)
├── tp-4/        # Mira API : PostgreSQL, enrichissement asynchrone, recherche hybride
└── tp-5/        # Serveur MCP : expose l'API tp-4 à un agent IA (Claude Code, Claude Desktop...)
```

`go-warmup/`, `tp-1/`, `tp-2/` et `tp-3/` sont des exercices d'apprentissage, non maintenus au-delà
de leur propre TP — chacun a son `notes.md` (notions apprises) et `commands.md` (commandes
d'exécution) si besoin d'y revenir.

## `tp-4` + `tp-5` — le projet Mira

`tp-4` et `tp-5` sont liés : **`tp-5` ne fonctionne pas sans `tp-4`**.

- **`tp-4/api`** est le serveur qui fait le travail réel : stockage PostgreSQL (pgx, migrations
  golang-migrate appliquées automatiquement au démarrage), enrichissement automatique et asynchrone
  (tags/résumé/score/embedding simulés localement, pool de workers borné avec timeout `context`,
  statut `enrichment_status` `pending`/`done`/`failed`), recherche hybride full-text + similarité
  vectorielle (pgvector). Il expose `/api/v1/notes` et `/api/v1/search` en HTTP.
  `tp-4/cli` est un client en ligne de commande pour cette même API.
- **`tp-5/cmd/mira-mcp`** est un serveur [MCP](https://modelcontextprotocol.io/) (transport stdio,
  `modelcontextprotocol/go-sdk`) qui traduit les appels d'un agent IA (Claude Code, Claude Desktop)
  en requêtes HTTP vers `tp-4/api` : 4 tools — `search_notes`, `get_note`, `add_note`,
  `list_recent_notes`. Comme `tp-4/cli`, il ne parle **jamais** à la base de données directement,
  uniquement à l'API HTTP — c'est ce qui garantit que toute note créée par l'agent déclenche
  l'enrichissement automatique.

Concrètement, `tp-5` est un adaptateur sans valeur seul : si `tp-4/api` n'est pas démarrée, ses
tools répondent avec une erreur claire ("mira API unreachable"), jamais un crash.

### Lancer le projet depuis un clone du dépôt

Prérequis : Go 1.26+, Docker Desktop (pour PostgreSQL/pgvector), Claude Code (pour le serveur MCP).

**1. Cloner et récupérer les dépendances**

```bash
git clone https://github.com/remytreflest/GO mira
cd mira
go mod download
```

**2. Démarrer PostgreSQL (pgvector)**

```bash
docker compose -f tp-4/docker-compose.yml up -d
```

**3. Démarrer l'API `tp-4`** — les migrations s'appliquent automatiquement au démarrage ; laisser ce
terminal ouvert, c'est un serveur qui doit rester en vie :

```bash
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api
```

Vérifier qu'elle répond :

```bash
curl http://localhost:8080/api/v1/notes?limit=1
```

**4. (Optionnel) Utiliser le CLI directement, sans agent IA**

```bash
go run ./tp-4/cli
```

**5. Enregistrer le serveur MCP `tp-5` dans Claude Code**

```bash
cp tp-5/.mcp.json.example .mcp.json
```

Redémarrer Claude Code (ou `/mcp` pour recharger les serveurs du projet). Contrairement à
`tp-4/api`, **`tp-5/cmd/mira-mcp` ne se lance jamais manuellement** : c'est Claude Code qui démarre
le binaire lui-même en sous-processus (transport stdio) dès qu'il en a besoin.

**6. Vérifier que tout fonctionne**

Demander à l'agent, par exemple : *"Ajoute une note résumant ce qu'on vient de faire"* — l'appel à
`add_note` doit apparaître dans les outils utilisés, et la note doit apparaître enrichie côté API
quelques centaines de ms plus tard :

```bash
curl http://localhost:8080/api/v1/notes/<id>
# enrichment_status: "done"
```

Voir [tp-4/notes.md](tp-4/notes.md) / [tp-4/commands.md](tp-4/commands.md) et
[tp-5/README.md](tp-5/README.md) / [tp-5/notes.md](tp-5/notes.md) /
[tp-5/commands.md](tp-5/commands.md) pour le détail de chaque partie.
