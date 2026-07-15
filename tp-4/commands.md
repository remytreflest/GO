# TP-4 — Commandes

Voir `api/commands.md` et `cli/commands.md` pour le détail. Résumé du scénario complet :

```bash
# 1. Démarrer Postgres (pgvector), port hôte 5433
docker compose -f tp-4/docker-compose.yml up -d

# 2. Démarrer l'API (migrations appliquées automatiquement)
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api

# 3. Dans un autre terminal, utiliser le CLI (parle à l'API en HTTP)
go run ./tp-4/cli add -tags go,concurrency "Go concurrency" "Goroutines, channels, worker pools"
go run ./tp-4/cli get <id>              # enrichment_status: pending, puis done peu après
go run ./tp-4/cli search Goroutines      # recherche hybride côté serveur
go run ./tp-4/cli update -content "..." <id>   # redéclenche l'enrichissement

# 4. Tests (offline, aucun Docker requis)
go test ./tp-4/...

# 5. Tests d'intégration Postgres (nécessitent l'étape 1)
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" \
  go test ./tp-4/api/internal/store/postgres/... -v

# 6. Arrêter Postgres
docker compose -f tp-4/docker-compose.yml down
```
