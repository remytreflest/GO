# TP-2 — Mira API v1

## Routeur `net/http` avec méthode + pattern (Go 1.22+)

```go
mux.HandleFunc("GET /api/v1/notes/{id}", h.Get)
id := r.PathValue("id")
```

Depuis Go 1.22, `http.ServeMux` route sur `MÉTHODE PATTERN` et supporte les wildcards `{id}`. Plus besoin d'un routeur tiers (gorilla/mux, chi...) pour ce niveau de besoin. Bonus : si le path matche mais pas la méthode, le mux répond automatiquement `405 Method Not Allowed`.

## Enveloppe de réponse stable

```go
type envelope struct {
    Data      any        `json:"data,omitempty"`
    Error     *errorBody `json:"error,omitempty"`
    RequestID string     `json:"request_id,omitempty"`
}
```

**Pourquoi :** un client de l'API n'a qu'une seule forme à parser, que la requête réussisse ou échoue. Évite les réponses "nues" incohérentes (tantôt un objet, tantôt un tableau, tantôt une string d'erreur).

## Validation explicite multi-champs

```go
ve := core.NewValidationError()
if title == "" {
    ve.Add("title", "title is required")
}
if ve.HasErrors() {
    return ve
}
```

Plutôt que de retourner à la première erreur trouvée (`errors.New("title required")`), on accumule toutes les erreurs de validation dans une map `field -> message`. Le client corrige tout en un seul aller-retour au lieu de découvrir les erreurs une par une.

## Pointeurs pour distinguer "absent" de "zéro" dans un PATCH

```go
type UpdateNoteInput struct {
    Title   *string   `json:"title,omitempty"`
    Content *string   `json:"content,omitempty"`
    Tags    *[]string `json:"tags,omitempty"`
}
```

Avec un `string` simple, impossible de savoir si le client veut vider le titre (`""`) ou n'a simplement pas touché ce champ. Un `*string` nil = champ absent du JSON ; un `*string` non-nil pointant sur `""` = le client a explicitement envoyé une chaîne vide (qu'on rejette ensuite en validation).

## `json.Decoder` strict

```go
dec := json.NewDecoder(r.Body)
dec.DisallowUnknownFields()
if err := dec.Decode(&v); err != nil { ... }
if dec.More() { /* deux objets JSON dans le corps */ }
```

`DisallowUnknownFields` rejette un payload avec un champ non attendu (`{"titre": "..."}` au lieu de `title`) au lieu de l'ignorer silencieusement. `dec.More()` détecte un second objet JSON concaténé au premier, que `Decode` seul laisserait passer.

## Pattern repository : `core.Store` (interface) / `store.MemoryStore` (implémentation)

```go
// core/store.go
type Store interface {
    Create(n *Note) error
    Update(id string, patch UpdateNoteInput) (*Note, error)
    ...
}
```

Comme dans tp-1 (`notes.Store` / `JSONLStore`), les handlers ne connaissent que l'interface. `MemoryStore` peut être remplacé par une implémentation SQL plus tard sans toucher à `internal/http/handlers`. `Update` prend le patch en paramètre (plutôt que Get+modify+Save côté handler) pour que le store applique la mutation sous un seul verrou — évite une race entre deux PATCH concurrents sur la même note.

## `sync.RWMutex` pour un store concurrent-safe

```go
type MemoryStore struct {
    mu    sync.RWMutex
    notes map[string]*core.Note
}
```

`RLock`/`RUnlock` pour les lectures (Get, List, Search) qui peuvent s'exécuter en parallèle entre elles ; `Lock`/`Unlock` pour les écritures (Create, Update, Delete) qui doivent être exclusives. `Get`/`List`/`Search` retournent des **copies** des notes (`cp := *n`) : sans ça, un appelant qui modifie la note reçue modifierait directement l'entrée du map, en dehors de tout verrou.

## Chaîne de middlewares par composition de `http.Handler`

```go
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
    for i := len(mw) - 1; i >= 0; i-- {
        h = mw[i](h)
    }
    return h
}
```

Chaque middleware est `func(http.Handler) http.Handler` : il reçoit le handler suivant et retourne un nouveau handler qui fait son travail puis appelle `next.ServeHTTP`. `Chain(router, RequestID, Logging, Recovery, Timeout)` construit `RequestID(Logging(Recovery(Timeout(router))))` : RequestID s'exécute en premier (les autres en ont besoin), Recovery est à l'intérieur de Logging pour que le statut 500 d'un panic récupéré soit bien loggé.

## Propager le request ID via `context.Context`

```go
ctx := context.WithValue(r.Context(), requestIDKey, id)
next.ServeHTTP(w, r.WithContext(ctx))
```

Le middleware `RequestID` injecte l'ID dans le contexte de la requête. Les handlers et le middleware `Logging` le relisent avec `middleware.RequestIDFromContext(ctx)`, sans avoir à le faire transiter par un paramètre de fonction supplémentaire à travers toutes les couches.

## Logging structuré avec `log/slog`

```go
logger.Info("http_request",
    "request_id", requestID,
    "method", r.Method,
    "status", rec.status,
    "duration_ms", time.Since(start).Milliseconds(),
)
```

`slog` (stdlib depuis Go 1.21) produit des paires clé/valeur exploitables par un agrégateur de logs (JSON via `slog.NewJSONHandler`), contrairement à `log.Printf` qui produit du texte libre à parser.

## Capturer le status HTTP réellement écrit

```go
type statusRecorder struct {
    http.ResponseWriter
    status int
}
func (rec *statusRecorder) WriteHeader(status int) {
    rec.status = status
    rec.ResponseWriter.WriteHeader(status)
}
```

`http.ResponseWriter` ne permet pas de relire le status code après coup. On enveloppe le writer pour intercepter `WriteHeader` et mémoriser la valeur, uniquement pour les besoins du log.

## `recover()` dans un middleware plutôt que dans chaque handler

```go
defer func() {
    if err := recover(); err != nil {
        logger.Error("panic_recovered", "error", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}()
```

Un seul point de récupération pour tous les handlers : un panic dans n'importe quel handler (nil pointer, index out of range...) devient un 500 propre au lieu de faire crasher tout le processus `net/http`.

## `http.TimeoutHandler`

```go
http.TimeoutHandler(next, 5*time.Second, `{"error":{"code":"timeout",...}}`)
```

Fait partie de la stdlib : si le handler suivant n'a pas fini d'écrire une réponse avant le délai, `TimeoutHandler` répond lui-même `503 Service Unavailable` avec le corps donné et abandonne le handler en cours (celui-ci continue de tourner en arrière-plan mais son écriture est ignorée).

## Tester les branches d'erreur "impossibles" avec un store bouchon

```go
type errStore struct{}
func (errStore) Create(*core.Note) error { return errBoom }
```

`MemoryStore` ne renvoie jamais d'erreur générique (seulement `ErrNotFound`), donc les branches `500 internal_error` des handlers ne sont jamais déclenchées en pratique. Pour les couvrir quand même (règle de couverture 100%), on implémente `core.Store` avec un stub qui échoue systématiquement et on route les handlers dessus dans un test dédié.

## Génération de la spec OpenAPI via annotations (`swaggo/swag`)

```go
//	@Summary	Créer une note
//	@Param		note	body		core.CreateNoteInput	true	"Note à créer"
//	@Success	201		{object}	envelope{data=core.Note}
//	@Router		/api/v1/notes [post]
func (h *NotesHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
```

La spec (`docs/swagger.yaml`) est générée à partir de commentaires structurés au-dessus des handlers via `swag init` (voir `commands.md`), et non écrite à la main : elle ne peut pas diverger silencieusement du code tant qu'on pense à relancer la génération après une modification de route.
