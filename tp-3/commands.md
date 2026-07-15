# TP-3 — Commandes

```bash
# Build de tous les exercices
go build ./tp-3/...

# Exécuter un exercice donné
go run ./tp-3/exo1
go run ./tp-3/exo2
go run ./tp-3/exo3
go run ./tp-3/exo4
go run ./tp-3/exo5      # ~2,4 s (worker lent simulé)
go run ./tp-3/exo6/race # résultat instable, < 1000
go run ./tp-3/exo6/fixed
go run ./tp-3/bonus     # ~1 s (annulation via context.WithTimeout)

# Détecteur de race (nécessite un compilateur C / CGO_ENABLED=1)
go run -race ./tp-3/exo6/race
```

Voir `exoN/commands.md` (et `bonus/commands.md`) pour le détail des sorties
observées de chaque exercice.
