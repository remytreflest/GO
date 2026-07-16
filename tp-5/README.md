# TP-5 — Serveur MCP mira

Un serveur [MCP](https://modelcontextprotocol.io/) (Model Context Protocol) en transport **stdio**
qui expose les notes de mira à un agent IA (Claude Code, Claude Desktop, tout client MCP) : recherche
hybride, lecture, création, dernières notes. Construit avec le SDK officiel
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

Comme `tp-4/cli`, ce serveur ne parle **qu'à l'API HTTP de `tp-4/api`**, jamais à la base de données
en direct : c'est la seule manière de garantir que toute note créée par l'agent déclenche
l'enrichissement automatique.

## Structure

```
tp-5/
├── cmd/mira-mcp/         # binaire : construction du serveur, transport stdio, logs
├── internal/miraclient/  # client HTTP vers l'API mira (timeouts via context)
├── internal/tools/       # les 4 tools MCP (schémas, validation, appels API, erreurs propres)
├── .env.example          # MIRA_API_URL
└── .mcp.json.example     # config d'exemple pour Claude Code
```

## Tools exposés

| Tool | Paramètres | Rôle |
|---|---|---|
| `search_notes` | `query` (string, requis), `limit` (int, défaut 10, max 50) | Recherche hybride full-text + vectorielle. Retourne un aperçu (titre, résumé, tags, statut) — utiliser `get_note` pour le contenu complet. |
| `get_note` | `id` (string, requis) | Retourne une note complète (contenu, tags, résumé, statut d'enrichissement). |
| `add_note` | `title`, `content` (string, requis), `tags` (optionnel) | Crée une note. `enrichment_status` vaut `pending` à la réponse puis passe à `done` peu après (traitement asynchrone côté API). |
| `list_recent_notes` | `limit` (int, défaut 10, max 100) | Dernières notes créées, triées de la plus récente à la plus ancienne. |

Chaque tool valide ses entrées et retourne une erreur MCP propre (`IsError` + message clair) en cas
de problème — jamais de panic ni de trace brute. Chaque appel à l'API sous-jacente est borné par un
timeout `context` (10s par défaut, voir `internal/miraclient.DefaultTimeout`).

## Installation

Depuis la racine du module (`mira/`) :

```bash
go build -o tp-5/mira-mcp.exe ./tp-5/cmd/mira-mcp
```

Ou, pour du développement rapide sans binaire, `go run ./tp-5/cmd/mira-mcp` (utilisé directement
dans la config Claude Code ci-dessous).

## Configuration

| Variable | Défaut | Rôle |
|---|---|---|
| `MIRA_API_URL` | `http://localhost:8080` | URL de base de l'API mira (`tp-4/api`) |

Les logs applicatifs (démarrage, erreurs d'appel API, panics récupérés) vont sur **stderr** via
`log/slog` — jamais sur stdout, réservé au protocole JSON-RPC en transport stdio.

## Démarrer manuellement (sans agent)

```bash
docker compose -f tp-4/docker-compose.yml up -d
DATABASE_URL="postgres://mira:mira@localhost:5433/mira?sslmode=disable" go run ./tp-4/api/cmd/api

# dans un autre terminal, depuis la racine du repo :
go run ./tp-5/cmd/mira-mcp
```

Le serveur attend des messages JSON-RPC sur stdin — il n'a pas d'interface interactive ; c'est
normal qu'il ne produise rien tant qu'aucun client MCP ne lui parle.

## Enregistrement dans Claude Code

Claude Code lit un fichier `.mcp.json` à la racine du projet. Copier l'exemple fourni :

```bash
cp tp-5/.mcp.json.example .mcp.json
```

```json
{
  "mcpServers": {
    "mira": {
      "command": "go",
      "args": ["run", "./tp-5/cmd/mira-mcp"],
      "env": { "MIRA_API_URL": "http://localhost:8080" }
    }
  }
}
```

Redémarrer Claude Code (ou lancer `/mcp` pour recharger les serveurs du projet). Le serveur `mira`
et ses 4 tools doivent apparaître dans la liste des outils MCP disponibles. L'API `tp-4/api` doit
être démarrée avant d'utiliser les tools (sinon les appels échouent proprement avec un message
d'erreur explicite, sans crash du serveur MCP).

## Enregistrement dans Claude Desktop

Claude Desktop ne connaît pas le répertoire du projet : construire d'abord le binaire, puis
référencer son chemin absolu dans `claude_desktop_config.json`
(`%APPDATA%\Claude\claude_desktop_config.json` sous Windows,
`~/Library/Application Support/Claude/claude_desktop_config.json` sous macOS) :

```bash
go build -o tp-5/mira-mcp.exe ./tp-5/cmd/mira-mcp
```

```json
{
  "mcpServers": {
    "mira": {
      "command": "C:\\chemin\\absolu\\vers\\mira\\tp-5\\mira-mcp.exe",
      "env": { "MIRA_API_URL": "http://localhost:8080" }
    }
  }
}
```

Redémarrer Claude Desktop pour charger le nouveau serveur.

## Exemples de prompts

Une fois l'API démarrée et le serveur MCP enregistré et rechargé :

- *"Retrouve ma note sur les channels Go"* → l'agent appelle `search_notes`.
- *"Ajoute une note résumant ce qu'on vient de faire, avec le tag mcp"* → l'agent appelle `add_note`.
- *"Montre-moi les 5 dernières notes que j'ai prises"* → l'agent appelle `list_recent_notes`.
- *"Donne-moi le contenu complet de la note <id>"* → l'agent appelle `get_note`.

## Tests

```bash
go test ./tp-5/... -cover
```

`internal/miraclient` est testé offline via `httptest` (aucune API requise).
`internal/tools` et `cmd/mira-mcp` sont testés bout en bout via les transports en mémoire du SDK
(`mcp.NewInMemoryTransports`), qui font transiter de vrais messages `tools/list` et `tools/call` —
le même chemin qu'un agent MCP réel emprunterait.

## Vérification manuelle de bout en bout

Voir `commands.md` pour le scénario complet (démarrage API + serveur MCP + appel via l'inspecteur
MCP ou un agent).
