# TP-4 API — Commandes

Toutes les commandes ci-dessous sont à lancer depuis la racine du module (`mira/`), sauf mention
contraire.

## Démarrer PostgreSQL

```bash
docker compose -f tp-4/docker-compose.yml up -d
```

Utilise l'image `pgvector/pgvector:pg16`, exposée sur le port hôte **5433** (pas 5432, pour éviter
tout conflit avec un autre Postgres déjà lancé localement). Arrêter :

```bash
docker compose -f tp-4/docker-compose.yml down       # garde les données
docker compose -f tp-4/docker-compose.yml down -v    # supprime aussi le volume
```

## Lancer le serveur

```bash
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api
```

Les migrations s'appliquent automatiquement à chaque démarrage (idempotent : `migrate.ErrNoChange`
est traité comme un succès). Variables d'environnement disponibles (voir `tp-4/.env.example`) :

| Variable | Défaut | Rôle |
|---|---|---|
| `DATABASE_URL` | DSN local docker-compose (port 5433) | connexion Postgres |
| `PORT` | `8080` | port d'écoute HTTP |
| `ENRICHMENT_WORKERS` | `4` | nombre de goroutines du pool d'enrichissement |
| `ENRICHMENT_QUEUE_SIZE` | `128` | taille du buffer du channel de jobs |
| `ENRICHMENT_JOB_TIMEOUT` | `3s` | timeout de l'écriture `SaveEnrichment` par job |

## Build

```bash
go build -o tp-4/api/api.exe ./tp-4/api/cmd/api
```

## Tests

Tests offline (aucun Docker requis — c'est la majorité de la suite) :

```bash
go test ./tp-4/api/...
```

Avec couverture :

```bash
go test ./tp-4/api/... -cover
```

Tests d'intégration Postgres (`internal/store/postgres/*_integration_test.go`, `t.Skip()` sans
`DATABASE_URL`) — nécessitent le `docker-compose.yml` ci-dessus démarré :

```bash
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" \
  go test ./tp-4/api/internal/store/postgres/... -v
```

## Vérifications statiques

```bash
go vet ./tp-4/api/...
gofmt -l ./tp-4/api
```

## (Re)générer la spec OpenAPI

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -d ./tp-4/api --output tp-4/api/docs --parseDependency --parseInternal
```

**À relancer systématiquement après toute modification d'une route, d'un payload ou d'un code de
réponse** (y compris les nouveaux champs `enrichment_status`/`summary`/`score`).

## Tester la spec dans un navigateur (Swagger UI)

Serveur lancé, ouvrir :

```
http://localhost:8080/swagger/index.html
```

## Vérification manuelle de bout en bout

```bash
# 1. Postgres
docker compose -f tp-4/docker-compose.yml up -d

# 2. API
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api

# 3. Dans un autre terminal : créer une note (répond immédiatement, enrichment_status=pending)
curl -X POST localhost:8080/api/v1/notes -H "Content-Type: application/json" \
  -d '{"title":"Go generics","content":"type parameters, constraints, generic functions"}'

# 4. Reconsulter juste après : enrichment_status doit être passé à "done" (tags/summary/score remplis)
curl localhost:8080/api/v1/notes/<id>

# 5. Recherche hybride
curl "localhost:8080/api/v1/search?q=goroutine%20channels"
```

Voir `tp-4/commands.md` pour le scénario complet incluant le CLI.
