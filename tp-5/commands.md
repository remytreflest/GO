# TP-5 — Commandes

Toutes les commandes se lancent depuis la racine du module (`mira/`).

## 1. Démarrer l'API mira (dépendance du serveur MCP)

```bash
docker compose -f tp-4/docker-compose.yml up -d
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api
```

## 2. Build / run du serveur MCP

```bash
go build -o tp-5/mira-mcp.exe ./tp-5/cmd/mira-mcp   # binaire
go run ./tp-5/cmd/mira-mcp                          # ou directement, sans build
```

Le serveur ne produit rien tant qu'aucun client MCP ne lui envoie de messages JSON-RPC sur stdin —
c'est le comportement attendu, pas un blocage.

## 3. Tests (offline, aucune API ni Docker requis)

```bash
go test ./tp-5/...
go test ./tp-5/... -cover
go test ./tp-5/... -coverprofile=cov.out && go tool cover -func=cov.out
```

## 4. Enregistrer le serveur dans Claude Code

Pour cela, il faut lancer claude code à la racine du projet mira

```bash
cp tp-5/.mcp.json.example .mcp.json
```

Puis recharger les serveurs MCP du projet (`/mcp` dans Claude Code, ou redémarrer -> c'est mieux). Voir
`README.md` pour le détail de la configuration et l'enregistrement dans Claude Desktop.

## 5. Vérification manuelle de bout en bout (sans agent, via l'inspecteur MCP)

Avec l'API démarrée (étape 1) et le serveur MCP construit (étape 2), l'[inspecteur MCP
officiel](https://github.com/modelcontextprotocol/inspector) permet d'appeler les tools sans agent :

```bash
npx @modelcontextprotocol/inspector go run ./tp-5/cmd/mira-mcp
```

Ouvrir l'URL affichée dans un navigateur, puis :

1. `add_note` avec `{"title": "Go channels", "content": "Notes sur les channels Go", "tags": ["go"]}`
   → répond avec `enrichment_status: "pending"`.
2. `get_note` avec l'`id` retourné, quelques centaines de ms plus tard → `enrichment_status: "done"`,
   `tags`/`summary`/`score` renseignés.
3. `search_notes` avec `{"query": "channels"}` → doit retrouver la note créée.
4. `list_recent_notes` → doit lister la note en premier (plus récente).

## 6. Démo live avec Claude Code (critère de validation du TP)

1. Étapes 1, 2, 4 ci-dessus.
2. Demander à Claude Code : *"Retrouve ma note sur les channels Go et ajoute une note résumant ce
   qu'on vient de faire"*.
3. Observer les appels `search_notes` puis `add_note` dans la liste des outils utilisés.
4. Vérifier côté API que la note créée est bien enrichie :

```bash
curl localhost:8080/api/v1/notes/<id>
# enrichment_status: "done" après quelques centaines de ms
```
