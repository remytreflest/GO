# TP-4 CLI — Commandes

Le CLI parle désormais à `tp-4/api` en HTTP — il faut l'API démarrée (voir `tp-4/api/commands.md`)
avant d'utiliser le CLI.

## Objectifs (évolution par rapport à tp-1)

- Remplacer le store JSONL local (`~/.mira/notes.jsonl`) par un client HTTP
  (`internal/apiclient.Client`), qui implémente la même interface `notes.Store`.
- Garantir que toute note créée/modifiée déclenche l'enrichissement asynchrone côté API.
- Ajouter une commande `update` (absente de tp-1) pour exercer PATCH → ré-enrichissement.
- La recherche (`search`) appelle directement l'endpoint hybride de l'API, plus de filtrage local.

## Configuration

| Variable | Défaut |
|---|---|
| `MIRA_API_URL` | `http://localhost:8080` |

```bash
export MIRA_API_URL=http://localhost:8080   # si l'API tourne sur un autre host/port
```

## Build

```bash
go build -o tp-4/cli/mira.exe ./tp-4/cli
```

## Commandes CLI

**Important — les flags précèdent toujours les positionnels** (`flag.Parse` s'arrête au premier
argument non-flag, voir `notes.md`) :

**Ajouter une note :**

```bash
go run ./tp-4/cli add "Go interfaces" "Les interfaces sont implicites en Go"
go run ./tp-4/cli add -tags api,rest "REST API" "GET POST PUT DELETE"
```

**Lister les 10 dernières notes :**

```bash
go run ./tp-4/cli list
```

**Rechercher (recherche hybride côté API) :**

```bash
go run ./tp-4/cli search go
go run ./tp-4/cli search "goroutine channels"
```

**Lire une note complète (avec statut d'enrichissement, résumé, score) :**

```bash
go run ./tp-4/cli get <id>
```

**Mettre à jour une note (redéclenche l'enrichissement) :**

```bash
go run ./tp-4/cli update -content "nouveau contenu" <id>
go run ./tp-4/cli update -title "Nouveau titre" -tags go,api <id>
```

**Supprimer une note :**

```bash
go run ./tp-4/cli delete <id>
```

## Mode interactif (sans arguments)

```bash
go run ./tp-4/cli
```

```
=== Mira — mode interactif (API) ===
> ajouter | lire | modifier | supprimer | lister | rechercher | quitter
:
```

## Tests

```bash
go test ./tp-4/cli/...
go test ./tp-4/cli/... -cover
```

`internal/apiclient` est testé offline via `httptest` (aucune API ni Docker requis).

## Vérification manuelle de bout en bout

Avec `tp-4/api` démarré (voir `tp-4/api/commands.md`) :

```bash
go run ./tp-4/cli add -tags go,concurrency "Go concurrency" "Goroutines, channels, worker pools"
# -> Added [<id>] Go concurrency (enrichment: pending)

go run ./tp-4/cli get <id>
# quelques centaines de ms plus tard : enrichment_status = done, tags/summary/score renseignés

go run ./tp-4/cli search goroutine
# -> passe par /api/v1/search, pas de filtrage local

go run ./tp-4/cli update -content "nouveau contenu, plus long" <id>
# -> enrichment_status repasse à pending puis done
```

Confirmer que `%USERPROFILE%\.mira\notes.jsonl` n'est ni créé ni modifié par ces commandes : le CLI
tp-4 ne touche plus jamais ce fichier.
