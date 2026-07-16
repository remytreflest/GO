# TP-5 — Serveur MCP mira

Notions clés apprises en branchant mira sur le Model Context Protocol via
`modelcontextprotocol/go-sdk`.

## MCP en une phrase

JSON-RPC 2.0 sur un transport donné (ici **stdio** : stdin/stdout), avec trois primitives côté
serveur — *tools* (actions appelables), *resources* (données lisibles), *prompts* (templates). Ce
TP n'utilise que les tools. Un client (Claude Code, Claude Desktop, un IDE) découvre les tools via
`tools/list` puis les appelle via `tools/call`.

## `mcp.NewServer` + `mcp.AddTool` — l'essentiel du SDK

```go
server := mcp.NewServer(&mcp.Implementation{Name: "mira", Version: "0.1.0"}, &mcp.ServerOptions{
    Logger: logger, // active le logging d'activité serveur
})

mcp.AddTool(server, &mcp.Tool{
    Name:        "search_notes",
    Description: "...", // fait partie du contrat : l'agent choisit son tool en la lisant
}, handlerFunc)

server.Run(ctx, &mcp.StdioTransport{})
```

`AddTool` est générique : `handlerFunc` a la signature
`func(ctx, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)`. Le SDK se charge de :

- dériver le schéma JSON d'entrée/sortie par réflexion sur `In`/`Out`,
- désérialiser et **valider** les arguments reçus contre ce schéma avant d'appeler le handler,
- sérialiser `Out` en `StructuredContent` (et en texte JSON dans `Content` si non renseigné).

## Schéma JSON dérivé des tags Go — la règle qui compte

Package `github.com/google/jsonschema-go` (dépendance du SDK) :

```go
type SearchNotesInput struct {
    Query string `json:"query" jsonschema:"terme à rechercher"`      // requis
    Limit int    `json:"limit,omitempty" jsonschema:"défaut 10"`     // optionnel
}
```

- **`omitempty`/`omitzero` absent → champ requis** dans le schéma JSON (`required: [...]`).
- **`omitempty` présent → champ optionnel**. Pas de mot-clé `default` supporté par le tag : le
  handler applique lui-même la valeur par défaut si le champ vaut zero-value (`clamp()` dans
  `internal/tools/tools.go`).
- Le tag `jsonschema:"..."` devient la `Description` de la propriété — c'est ce que l'agent lit
  pour comprendre le paramètre. Pas de mini-langage de contraintes (min/max/etc.) via ce tag dans
  cette version du SDK ; possible seulement en construisant le schéma à la main.

## Erreur de handler → `IsError`, jamais une exception protocole

Point le plus important pour le critère "jamais de panic, jamais de stack trace brute" : avec
`AddTool` (API générique `ToolHandlerFor`), un `error` non-nil retourné par le handler est
**automatiquement empaqueté** dans `CallToolResult{IsError: true, Content: [...]}` — ce n'est pas
une erreur JSON-RPC protocole (contrairement à l'API bas niveau `ToolHandler`, cf. doc du SDK). Le
client MCP reçoit donc toujours une réponse structurée, jamais un crash de session.

Ça ne couvre pas les vrais `panic()` Go (bug interne, nil deref...) : rien dans le SDK ne les
récupère avant qu'ils ne remontent jusqu'à la boucle de dispatch. D'où `recovered()` dans
`internal/tools/tools.go` — un middleware générique qui wrap chaque `ToolHandlerFor`, capture un
panic éventuel via `recover()`, le logue sur stderr et le convertit en erreur propre. Sans ça, un
bug dans un handler tuerait le process stdio entier et toutes les autres sessions en cours.

## Pourquoi passer par l'API HTTP et jamais par la base

Même contrainte que `tp-4/cli` (voir `tp-4/cli/notes.md`) : c'est l'API qui déclenche
l'enrichissement asynchrone (`h.Enricher.Submit(...)` dans les handlers HTTP). Un accès direct à la
base contournerait ce pipeline — la note resterait sans tags/résumé/score. `internal/miraclient`
est donc une copie allégée de `tp-4/cli/internal/apiclient`, adaptée au besoin de l'agent (pas de
`Update`/`Delete`, pas de pagination `All()` — juste les 4 opérations utilisées par les tools).

## Timeout par appel via `context`

Chaque méthode de `miraclient.Client` reçoit un `ctx` (celui du tool call) et l'enveloppe dans son
propre `context.WithTimeout` avant de construire la requête HTTP (`http.NewRequestWithContext`) :

```go
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
    ctx, cancel := context.WithTimeout(ctx, c.timeout())
    defer cancel()
    req, _ := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
    ...
}
```

Ainsi une API qui ne répond pas ne bloque jamais un tool call indéfiniment — le client reçoit
`context.DeadlineExceeded`, transformé en erreur MCP propre.

## Tester un serveur MCP sans process ni stdin/stdout réels

`mcp.NewInMemoryTransports()` fournit une paire de transports connectés en mémoire : on peut donc
démarrer un vrai `*mcp.Server`, un vrai `*mcp.Client`, les connecter, et échanger de vrais messages
`tools/list`/`tools/call` dans un test unitaire Go — sans lancer de binaire ni sérialiser sur un
pipe. C'est le moyen le plus fidèle de tester `Register()` : ça exerce l'inférence de schéma, la
validation d'entrée du SDK et le chemin complet, pas seulement la logique métier du handler appelée
directement.

## Recherche : `limit` géré côté client, pas côté API

`GET /api/v1/search` (tp-4/api) n'accepte pas de paramètre `limit` — elle retourne déjà un
top-K trié côté serveur (`searchResultLimit = 20` en SQL, voir
`tp-4/api/internal/store/postgres/search.go`). Plutôt que de modifier tp-4 (figé, cf. README
racine), `miraclient.Client.Search` tronque simplement les résultats côté client au `limit` demandé
par l'agent.
