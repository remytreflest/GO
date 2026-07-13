# TP-2 — Mira API v1

API HTTP `/api/v1/notes` du projet fil rouge Mira. Stockage en mémoire (map + mutex), aucune dépendance externe pour le runtime.

---

## Structure

```
tp-2/
├── cmd/api/
│   ├── main.go            # point d'entrée : wiring store/router/middlewares + ListenAndServe
│   └── main_test.go
├── internal/
│   ├── core/               # domaine : Note, validation, erreurs, interface Store
│   ├── store/               # implémentation Store en mémoire (map + sync.RWMutex)
│   └── http/
│       ├── handlers/       # handlers HTTP, enveloppe JSON, routeur
│       └── middleware/     # request ID, logging (slog), recovery, timeout
├── docs/                    # spec OpenAPI (Swagger 2.0) générée par swag — voir commands.md
├── cover                    # rapport de couverture de tests (go tool cover)
├── notes.md
└── commands.md
```

---

## Lancer le serveur

Depuis la racine du module (`mira/`) :

```bash
go run ./tp-2/cmd/api
```

Le serveur écoute sur `:8080` par défaut (`PORT=9090 go run ./tp-2/cmd/api` pour changer de port).

---

## Routes

| Méthode | Route | Description |
|---|---|---|
| POST | `/api/v1/notes` | Créer une note |
| GET | `/api/v1/notes?limit=&offset=` | Lister les notes (paginé) |
| GET | `/api/v1/notes/{id}` | Récupérer une note |
| PATCH | `/api/v1/notes/{id}` | Mettre à jour partiellement une note |
| DELETE | `/api/v1/notes/{id}` | Supprimer une note |
| GET | `/api/v1/search?q=...` | Recherche texte (titre + contenu, insensible à la casse) |

### Enveloppe de réponse

Toutes les réponses (succès et erreurs) suivent la même forme stable :

```json
{ "data": { ... }, "request_id": "a1b2c3d4e5f60718" }
```

```json
{ "error": { "code": "validation_error", "message": "invalid note payload", "fields": { "title": "title is required" } }, "request_id": "a1b2c3d4e5f60718" }
```

`request_id` est repris de l'en-tête `X-Request-ID` de la requête si présent, sinon généré. Il est aussi renvoyé dans l'en-tête de réponse `X-Request-ID`.

---

## Exemples `curl`

**Créer une note :**

```bash
curl -X POST localhost:8080/api/v1/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Go interfaces","content":"Les interfaces sont implicites","tags":["go"]}'
```

**Lister (paginé) :**

```bash
curl "localhost:8080/api/v1/notes?limit=10&offset=0"
```

**Récupérer une note :**

```bash
curl localhost:8080/api/v1/notes/<id>
```

**Mettre à jour partiellement :**

```bash
curl -X PATCH localhost:8080/api/v1/notes/<id> \
  -H "Content-Type: application/json" \
  -d '{"title":"Go interfaces (v2)"}'
```

**Supprimer :**

```bash
curl -X DELETE localhost:8080/api/v1/notes/<id>
```

**Rechercher :**

```bash
curl "localhost:8080/api/v1/search?q=go"
```

---

## Codes d'erreur possibles

| Status | Code | Quand |
|---|---|---|
| 400 | `invalid_body` | JSON malformé, corps vide, champ inconnu, ou plusieurs objets JSON |
| 400 | `validation_error` | Titre manquant/vide/trop long, contenu trop long, tag vide/trop long, PATCH sans aucun champ |
| 400 | `invalid_query` | `limit`/`offset` non numériques ou négatifs, `q` manquant sur `/search` |
| 404 | `not_found` | Note inexistante (GET/PATCH/DELETE) |
| 405 | — | Méthode HTTP non supportée sur une route existante (géré par `net/http.ServeMux`) |
| 500 | `internal_error` | Échec inattendu du store |
| 503 | `timeout` | Requête non terminée avant le timeout serveur (5s) |

---

## Pagination (bonus)

`GET /api/v1/notes?limit=&offset=` : `limit` par défaut 20, plafonné à 100 ; `offset` par défaut 0. La réponse inclut `total`, `limit` et `offset` pour construire la page suivante.

## Spec OpenAPI (bonus)

Générée depuis les annotations swaggo sur les handlers via l'outil `swag` — voir `commands.md` pour la commande de (ré)génération. Fichiers produits dans `docs/` (`swagger.json`, `swagger.yaml`, `docs.go`).

**Règle du projet : toute modification d'une route API doit s'accompagner d'une régénération de cette spec.**

### Tester la spec dans un navigateur (Swagger UI)

Le serveur expose une UI Swagger interactive tant qu'il tourne (`go run ./tp-2/cmd/api`) :

```
http://localhost:8080/swagger/index.html
```

Depuis cette page, chaque route peut être documentée, appelée et testée directement ("Try it out") sans passer par `curl`. Le JSON brut de la spec est servi sur `http://localhost:8080/swagger/doc.json`.
