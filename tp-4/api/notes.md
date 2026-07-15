# TP-4 — Mira API v2 (PostgreSQL, enrichissement, recherche hybride)

Évolution de `tp-2/`. Les sections ci-dessous, reprises de `tp-2/notes.md`, restent valables telles
quelles (routing, enveloppe, validation, middlewares, pattern repository, swaggo). Les nouveautés
propres au TP4 (Postgres/pgx, enrichissement asynchrone, recherche hybride) sont regroupées à la fin
du fichier, après la section swaggo.

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

Aucun plugin d'éditeur n'est impliqué : deux briques distinctes de l'écosystème `swaggo`, aucune installée globalement.

- **`swag` (outil CLI, générateur)** — invoqué via `go run github.com/swaggo/swag/cmd/swag@latest init ...` (`commands.md`). `go run module@latest` télécharge et exécute l'outil à la volée depuis le cache des modules Go, sans installation globale ni ajout au `PATH`. Il lit les commentaires `// @Summary`, `// @Param`, etc. au-dessus des handlers (`internal/http/handlers/notes.go`) et de `cmd/api/main.go`, puis génère `docs/swagger.json`, `docs/swagger.yaml` et `docs/docs.go`.
- **`swaggo/http-swagger` (bibliothèque, dépendance de runtime)** — déclarée dans `go.mod` (`github.com/swaggo/http-swagger/v2`) car importée directement dans le code : `main.go` l'importe sous l'alias `httpSwagger` et monte `mux.Handle("/swagger/", httpSwagger.WrapHandler)`. C'est elle qui sert l'UI Swagger à l'exécution, en lisant `docs.go` généré par `swag init`.

---

# Nouveautés TP-4 : PostgreSQL, enrichissement asynchrone, recherche hybride

## PostgreSQL via pgx, sans ORM

`internal/store/postgres.Store` implémente `core.Store` et `core.EnrichmentStore` avec
`github.com/jackc/pgx/v5/pgxpool` (pool de connexions) — pas d'ORM, du SQL explicite dans
`notes.go`/`search.go`/`enrichment.go`.

`core.Store` est devenu **ctx-aware** (`Create(ctx, n)`, `Get(ctx, id)`, ...) : nécessaire pour que
chaque requête HTTP puisse propager son annulation/timeout jusqu'à la requête SQL sous-jacente.
`tp-2/internal/core.Store` (figé) ne l'était pas ; c'est un changement délibéré propre à `tp-4`.

### Transaction sur création de note + tags

```go
tx, _ := s.pool.Begin(ctx)
defer tx.Rollback(ctx)  // no-op si Commit a déjà réussi

tx.Exec(ctx, `INSERT INTO notes (...) VALUES (...)`, ...)
tx.Exec(ctx, `INSERT INTO note_tags (note_id, tag) SELECT $1, unnest($2::text[])`, n.ID, n.Tags)

return tx.Commit(ctx)
```

`unnest($2::text[])` insère tous les tags en un seul aller-retour réseau plutôt qu'une requête par
tag. `defer tx.Rollback(ctx)` juste après `Begin` est le pattern idiomatique pgx : un rollback après
un commit réussi renvoie une erreur silencieusement ignorée, pas de panique.

### Une interface `EnrichmentStore` séparée, pas un `core.Store` plus gros

```go
type EnrichmentStore interface {
    SaveEnrichment(ctx context.Context, result EnrichmentResult) error
}
```

Plutôt que d'ajouter `SaveEnrichment` à `core.Store` (qui grossirait pour tous les appelants, y
compris ceux qui n'ont rien à voir avec l'enrichissement), c'est une interface à part, étroite.
`postgres.Store` et `store.MemoryStore` l'implémentent toutes les deux — ce qui permet de tester le
pool de workers et les handlers **entièrement offline**, sans Postgres.

## golang-migrate + `embed.FS`

Les migrations SQL (`migrations/000N_*.up.sql`/`.down.sql`) sont embarquées dans le binaire via
`//go:embed *.sql` (`migrations/embed.go`) et appliquées automatiquement au démarrage
(`cmd/api/migrate.go`), avant l'ouverture du pool applicatif.

Piège pgx : le driver Postgres de golang-migrate fonctionne sur `database/sql`, pas sur
`pgxpool.Pool`. Le démarrage ouvre donc une connexion `database/sql` **éphémère** (via
`github.com/jackc/pgx/v5/stdlib`, qui enregistre le driver `"pgx"`), l'utilise uniquement pour
`migrate.Up()`, la ferme, puis ouvre le `pgxpool.Pool` applicatif à part — deux connexions
distinctes, pour deux usages différents, vers le même DSN.

## Schéma & pgvector

- `notes` porte une colonne **générée** `search_vector tsvector` (config `'simple'`, pas
  `'french'`/`'english'` : l'expression d'une colonne générée doit être immuable, et `'simple'`
  garde un comportement déterministe sans stemming pour un corpus pédagogique bilingue) + un index
  **GIN** dessus.
- `note_embeddings.embedding` est un `vector(64)` (extension pgvector) avec un index **HNSW**
  (`vector_cosine_ops`) plutôt qu'ivfflat : ivfflat a besoin d'un paramètre `lists` calé sur la
  taille de la table et se dégrade sans ré-entraînement sur une petite table qui grossit
  incrémentalement (typiquement ce projet pédagogique) ; HNSW n'a pas ce problème.
- L'image Docker doit être `pgvector/pgvector:pg16`, pas `postgres:16` seul — l'extension
  `vector` n'est pas dans l'image officielle.
- Les IDs restent le schéma hex existant (`core.NewID()`, `crypto/rand`), pas des UUID : pas besoin
  de `pgcrypto`/`gen_random_uuid()`.
- `vector(64)` est codé en dur des deux côtés (la migration et `embedding.Dimension`) : pgvector ne
  redimensionne pas une colonne `vector` en place, changer la dimension impose une nouvelle
  migration et un reset de la table.

### Encodage des vecteurs sans codec pgx natif

`github.com/pgvector/pgvector-go` v0.4.0 n'implémente que `database/sql.Scanner`/`driver.Valuer`,
pas de codec binaire natif pour pgx v5. Plutôt que de dépendre d'un mécanisme d'enregistrement de
type incertain, `internal/store/postgres/vector.go` encode simplement le vecteur en son literal
texte pgvector (`"[0.1,0.2,...]"` via `pgvector.NewVector(v).String()`) et le passe comme paramètre
`string` avec un cast explicite `$N::vector` côté SQL. Ce choix a été vérifié contre un vrai
Postgres+pgvector (voir `postgres_integration_test.go`), pas seulement supposé correct.

## Enrichissement simulé, 100% local

`internal/embedding` ne fait aucun appel réseau/API externe — décision explicite pour rester
offline/déterministe/testable en CI, comme le reste des TP :

- `Embed(text) []float32` : hachage de sac-de-mots (FNV sur chaque token, bucket parmi
  `Dimension=64`), puis normalisation L2. Un texte vide utilise un token sentinelle pour ne jamais
  produire le vecteur nul (la similarité cosinus contre le vecteur nul n'est pas définie). Ce n'est
  **pas** un vrai embedding sémantique : deux textes proches lexicalement (mots partagés) se
  retrouvent proches, mais il n'y a aucune notion de synonyme ou de sens.
- `ExtractTags` : fréquence de mots hors liste de stop-words FR/EN, tri déterministe (l'itération
  de map en Go est aléatoire, donc le tri explicite par `(fréquence desc, mot asc)` est nécessaire).
- `Summarize` : première phrase si elle tient dans la limite, sinon troncature à la limite de mot +
  ellipse.
- `Score` : heuristique arbitraire (longueur + diversité du vocabulaire) — illustratif, pas un
  jugement de qualité.

## Pool de workers borné (`internal/enrichment`)

Reprend le pattern worker pool de `tp-3/exo4` et le `context.WithTimeout` de `tp-3/bonus`.

- `Submit(job) bool` : `select` non bloquant sur un channel bufferisé — la requête HTTP ne bloque
  **jamais** dessus (buffer plein → log + `false`, la note reste `pending`).
- Le timeout par job ne protège pas le calcul (pur, local, instantané) mais l'écriture
  `SaveEnrichment` (le seul point d'I/O — un aller-retour Postgres). En cas d'erreur/timeout sur
  cette écriture, un second essai **best-effort** avec un contexte frais marque la note `failed`
  plutôt que de la laisser bloquée à `pending` indéfiniment.
- `Shutdown` : ferme le channel `jobs` sous le même mutex que `Submit` vérifie/utilise — ce qui
  élimine la panique classique "send on closed channel" si `Submit` et `Shutdown` s'exécutent en
  même temps (un channel `stopped` séparé + `select` aurait laissé une fenêtre de course ; un mutex
  partagé, non).
- Ordre d'arrêt dans `main.go` : `srv.Shutdown` (plus de nouvelles requêtes/soumissions) →
  `pool.Shutdown` (draine les jobs en cours) → fermeture du pool Postgres.

## Recherche hybride

`postgres.Store.Search` calcule l'embedding de la requête côté Go (`embedding.Embed(q)`), puis
combine dans une seule requête SQL :

- `ts_rank(search_vector, plainto_tsquery('simple', $1))` — pertinence texte,
- `1 - (embedding <=> $2::vector)` — similarité cosinus (pgvector),

`ORDER BY 0.6*text_rank + 0.4*vector_sim DESC`, avec un seuil minimal de similarité vectorielle
pour que la clause `OR` ne fasse pas matcher toutes les notes enrichies. Poids et seuil sont des
constantes nommées dans `search.go`, faciles à retoucher.

## Tests : ce qui tourne offline, ce qui a besoin de Postgres

`internal/embedding`, `internal/enrichment` (double `core.EnrichmentStore`), `store.MemoryStore`,
`internal/http/handlers` (double store + double `Enricher`), `cmd/api` (injection de dépendances
sur `buildHandler`) : 100% offline, aucun Docker requis.

`internal/store/postgres/*_integration_test.go` : `t.Skip()` si `DATABASE_URL` n'est pas défini —
jamais exécutés par défaut. Voir `commands.md` pour les lancer contre le `docker-compose.yml` du
projet.
