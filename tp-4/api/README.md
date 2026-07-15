# TP-4 — Mira API v2

Évolution de `tp-2/` : API HTTP `/api/v1/notes` du projet fil rouge Mira, désormais sur **PostgreSQL**
(via pgx) avec **enrichissement asynchrone** (tags/résumé/score/embedding simulés, calculés par un
pool de workers) et **recherche hybride** (full-text + similarité vectorielle).

---

## Structure

```
tp-4/api/
├── cmd/api/
│   ├── main.go            # wiring : migrations → pgxpool → store → pool → router → serveur
│   ├── config.go           # helpers d'env (PORT, DATABASE_URL, ENRICHMENT_*)
│   └── migrate.go          # applique les migrations embarquées au démarrage (golang-migrate)
├── migrations/             # SQL + embed.FS
├── docs/                   # spec OpenAPI (Swagger 2.0) générée par swag
└── internal/
    ├── core/               # domaine : Note (+ Summary/Score/EnrichmentStatus), Store (ctx-aware),
    │                       # EnrichmentStore, validation, erreurs
    ├── embedding/          # fonctions pures : Embed, ExtractTags, Summarize, Score (simulées, sans API externe)
    ├── enrichment/         # pool de workers borné qui consomme les jobs d'enrichissement
    ├── store/
    │   ├── memory.go       # MemoryStore (tests offline)
    │   └── postgres/       # Store PostgreSQL (pgx) : transactions, hybrid search
    └── http/
        ├── handlers/       # handlers HTTP, enveloppe JSON, routeur
        └── middleware/     # request ID, logging (slog), recovery, timeout
```

---

## Lancer le serveur

Démarrer PostgreSQL (voir `tp-4/docker-compose.yml`, port hôte **5433** pour éviter tout conflit
avec un autre Postgres local) :

```bash
docker compose -f tp-4/docker-compose.yml up -d
```

Puis, depuis la racine du module (`mira/`) :

```bash
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api
```

Les migrations s'appliquent automatiquement au démarrage. Le serveur écoute sur `:8080` par défaut
(`PORT=9090` pour changer). Voir `.env.example` à la racine de `tp-4/` pour toutes les variables.

---

## Routes

| Méthode | Route | Description |
|---|---|---|
| POST | `/api/v1/notes` | Créer une note (déclenche l'enrichissement asynchrone) |
| GET | `/api/v1/notes?limit=&offset=` | Lister les notes (paginé) |
| GET | `/api/v1/notes/{id}` | Récupérer une note |
| PATCH | `/api/v1/notes/{id}` | Mettre à jour partiellement une note (redéclenche l'enrichissement) |
| DELETE | `/api/v1/notes/{id}` | Supprimer une note |
| GET | `/api/v1/search?q=...` | Recherche hybride (full-text + similarité vectorielle) |

### Note enrichie

Chaque note porte désormais :

```json
{
  "id": "091ef222656c3e2f",
  "title": "Go generics",
  "content": "type parameters constraints and generic functions in go",
  "tags": ["constraints", "functions", "generic", "generics", "parameters"],
  "summary": "type parameters constraints and generic functions in go",
  "score": 0.4775,
  "enrichment_status": "done",
  "created_at": "2026-07-15T11:42:50.167515+02:00",
  "updated_at": "2026-07-15T11:42:50.167515+02:00"
}
```

`enrichment_status` vaut `pending` juste après un POST/PATCH (la réponse est immédiate), puis
`done` ou `failed` une fois le job traité par le pool de workers — généralement en quelques
dizaines à quelques centaines de millisecondes. `tags`/`summary`/`score` ne sont renseignés
qu'après un enrichissement réussi.

### Enveloppe de réponse

Inchangée par rapport à tp-2 :

```json
{ "data": { ... }, "request_id": "a1b2c3d4e5f60718" }
```

```json
{ "error": { "code": "validation_error", "message": "invalid note payload", "fields": { "title": "title is required" } }, "request_id": "a1b2c3d4e5f60718" }
```

---

## Exemples `curl`

**Créer une note (répond immédiatement avec `enrichment_status: "pending"`) :**

```bash
curl -X POST localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Go generics","content":"type parameters, constraints, generic functions"}'
```

**Reconsulter juste après pour observer le passage `pending` → `done` :**

```bash
curl localhost:8080/api/v1/notes/<id>
```

**Recherche hybride :**

```bash
curl "localhost:8080/api/v1/search?q=goroutine%20channels"
```

Les autres routes (`list`, `get`, `PATCH`, `delete`) sont inchangées par rapport à `tp-2/README.md`.

---

## Codes d'erreur possibles

Identiques à tp-2 (`invalid_body`, `validation_error`, `invalid_query`, `not_found`,
`internal_error`, `timeout` à 503) — voir `tp-2/README.md` pour le détail.

---

## Spec OpenAPI

Générée depuis les annotations swaggo — voir `commands.md` pour la commande de régénération.

**Règle du projet : toute modification d'une route API doit s'accompagner d'une régénération de
cette spec.**

```
http://localhost:8080/swagger/index.html
```
