# TP-2 — Commandes

Toutes les commandes ci-dessous sont à lancer depuis la racine du module (`mira/`), sauf mention contraire.

## Lancer le serveur

```bash
go run ./tp-2/cmd/api
```

Port personnalisé :

```bash
PORT=9090 go run ./tp-2/cmd/api
```

## Build

```bash
go build -o tp-2/api.exe ./tp-2/cmd/api
```

## Tests

Tous les tests du TP :

```bash
go test ./tp-2/...
```

Avec couverture (doit rester à 100% sur `core`, `store`, `handlers`, `middleware` — voir la règle de couverture dans `notes.md`) :

```bash
go test ./tp-2/... -cover
```

Rapport détaillé par fonction :

```bash
go test ./tp-2/... -coverprofile=tp-2/cover.out
go tool cover -func=tp-2/cover.out
```

## Vérifications statiques

```bash
go vet ./tp-2/...
gofmt -l ./tp-2
```

## (Re)générer la spec OpenAPI (bonus)

Utilise l'outil `swag` (swaggo) qui lit les annotations placées au-dessus des handlers dans `internal/http/handlers/notes.go` et de `cmd/api/main.go`. Aucune installation globale requise, `go run` télécharge l'outil à la volée :

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -d ./tp-2 --output tp-2/docs --parseDependency --parseInternal
```

Produit `tp-2/docs/swagger.json`, `tp-2/docs/swagger.yaml` et `tp-2/docs/docs.go` (ce dernier est importé par `cmd/api/main.go` pour servir la spec — ne pas le supprimer).

**À relancer systématiquement après toute modification d'une route, d'un payload ou d'un code de réponse.**

## Tester la spec dans un navigateur (Swagger UI)

Serveur lancé (`go run ./tp-2/cmd/api`), ouvrir :

```
http://localhost:8080/swagger/index.html
```

## Tester manuellement (exemples curl)

Voir la section "Exemples curl" du `README.md`.
