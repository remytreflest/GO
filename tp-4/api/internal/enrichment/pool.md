# `internal/enrichment` — pool de workers borné pour l'enrichissement asynchrone

Ce package fait tourner l'enrichissement d'une note (tags/summary/score/embedding, calculés par
`internal/embedding` — voir `embedding.md`) en arrière-plan, sur un nombre borné de goroutines,
pour qu'une requête HTTP `POST /notes` ou `PATCH /notes/{id}` réponde immédiatement sans attendre ce
calcul.

Ordre de lecture logique : `job.go` (la donnée transportée) → `pool.go` (le pool lui-même :
construction, cycle de vie, traitement d'un job) → `pool_test.go` (le comportement observable, en
particulier les cas limites).

## `job.go` — `Job`, la donnée transportée

```go
type Job struct {
    NoteID  string
    Title   string
    Content string
}
```

Un `Job` ne contient que ce qui est nécessaire pour calculer l'enrichissement : l'identifiant de la
note à mettre à jour, plus son titre/contenu actuels. Rien d'autre ne transite par le channel de
jobs.

## `pool.go` — `Pool`, le worker pool

```go
type Pool struct {
    jobs    chan Job
    store   core.EnrichmentStore
    workers int
    timeout time.Duration
    logger  *slog.Logger

    mu     sync.Mutex
    closed bool
    wg     sync.WaitGroup
}
```

- `jobs` : channel **bufferisé** (taille = `queueSize` passé à `NewPool`) qui fait office de file
  d'attente entre les handlers HTTP (producteurs) et les workers (consommateurs).
- `store core.EnrichmentStore` : interface étroite (`SaveEnrichment(ctx, result) error`), pas
  `core.Store` complet — voir `tp-4/api/notes.md`, section "Une interface `EnrichmentStore`
  séparée", pour pourquoi ce découpage existe.
- `closed` + `mu` : protègent contre l'envoi sur un channel déjà fermé quand `Submit` et `Shutdown`
  s'exécutent en même temps (détail plus bas).
- `wg sync.WaitGroup` : permet à `Shutdown` d'attendre que tous les workers aient fini de vider la
  file avant de rendre la main.

### `NewPool` / `Start` — construction et démarrage

```go
func NewPool(store core.EnrichmentStore, workers, queueSize int, timeout time.Duration, logger *slog.Logger) *Pool
func (p *Pool) Start(ctx context.Context)
```

`NewPool` construit juste la structure (channel, compteurs) sans rien démarrer. `Start` lance
`workers` goroutines identiques (`p.worker`), chacune bouclant sur `range p.jobs` jusqu'à ce que le
channel soit fermé. Séparer construction et démarrage permet de créer le `Pool` tôt (ex. injection
de dépendances dans `cmd/api`) et de ne le démarrer qu'une fois le reste de l'application prêt.

### `process` — calcul + écriture, avec un timeout ciblé

```go
func (p *Pool) process(ctx context.Context, job Job) {
    jobCtx, cancel := context.WithTimeout(ctx, p.timeout)
    ...
    result := core.EnrichmentResult{
        Tags:      embedding.ExtractTags(text, maxTags),
        Summary:   embedding.Summarize(job.Content, summaryMaxLen),
        Score:     embedding.Score(job.Title, job.Content),
        Embedding: embedding.Embed(text),
        Status:    core.EnrichmentDone,
    }
    if err := p.store.SaveEnrichment(jobCtx, result); err != nil {
        // best-effort : marque la note "failed" plutôt que de la laisser "pending" indéfiniment
    }
}
```

Point important : le `timeout` ne protège **pas** le calcul (les 4 fonctions d'`internal/embedding`
sont pures, locales, quasi instantanées) mais uniquement `SaveEnrichment` — le seul point d'I/O
(un aller-retour Postgres). Si cet appel échoue ou dépasse le délai, un **second essai
best-effort**, avec un contexte frais et un timeout court dédié (`fallbackSaveTimeout`), tente de
marquer la note `failed` — pour ne jamais la laisser bloquée à `pending` pour toujours si l'écriture
initiale a échoué.

### `Submit` — ne jamais bloquer l'appelant HTTP

```go
func (p *Pool) Submit(job Job) bool {
    select {
    case p.jobs <- job:
        return true
    default:
        return false
    }
}
```

Le `select`/`default` rend l'envoi **non bloquant** : si le buffer du channel est plein (trop de
jobs en attente), `Submit` ne bloque pas la requête HTTP appelante — elle log un avertissement,
renvoie `false`, et la note reste `pending` (elle ne sera pas enrichie, mais l'API répond quand
même). C'est un choix explicite de dégradation gracieuse plutôt que d'attente/erreur 500.

### `Shutdown` — arrêt propre, sans course

```go
func (p *Pool) Shutdown(ctx context.Context) error {
    p.mu.Lock()
    if !p.closed {
        p.closed = true
        close(p.jobs)
    }
    p.mu.Unlock()
    ...
}
```

Fermer le channel `jobs` et vérifier/écrire `closed` sous le **même mutex** que `Submit` utilise est
ce qui évite la panique classique "send on closed channel" : sans ce verrou partagé, un `Submit` et
un `Shutdown` concurrents pourraient laisser passer un envoi juste après la fermeture du channel.
Une alternative à base d'un channel `stopped` séparé + `select` dans `Submit` laisserait une fenêtre
de course ; le mutex partagé, non.

Une fois le channel fermé, `Shutdown` attend (`wg.Wait()`, exécuté dans une goroutine séparée pour
pouvoir le combiner à un `select`) que tous les workers aient fini de vider la file, borné par le
`ctx` passé en paramètre — s'il expire avant que les workers aient terminé, `Shutdown` renvoie
`ctx.Err()` (typiquement `context.DeadlineExceeded`) sans attendre indéfiniment.

Ordre d'arrêt utilisé dans `cmd/api/main.go` : `srv.Shutdown` (n'accepte plus de nouvelles requêtes
donc plus de nouveaux `Submit`) → `pool.Shutdown` (draine les jobs déjà en file) → fermeture du pool
Postgres. Dans cet ordre, aucune écriture d'enrichissement n'est perdue.

## `pool_test.go` — le comportement vérifié

Le store réel (Postgres) est remplacé par un `fakeEnrichmentStore` qui enregistre les appels reçus
et peut simuler un délai (`delay`) — ce qui permet de tester des scénarios de course/timeout de
façon déterministe, sans jamais toucher Postgres :

- `TestPool_ProcessesJob` : un `Submit` réussi finit par produire un résultat `EnrichmentDone` avec
  tags/summary/embedding remplis.
- `TestPool_SaveTimeoutFallsBackToFailed` : si `SaveEnrichment` est plus lent que `timeout`, une
  seule sauvegarde est enregistrée au final, avec le statut `EnrichmentFailed` (le premier essai,
  qui a expiré, n'est volontairement pas compté).
- `TestPool_SubmitNonBlockingWhenQueueFull` : pool **non démarré** (rien ne vide le channel) pour
  remplir son buffer de façon déterministe et vérifier que le second `Submit` échoue proprement.
- `TestPool_ShutdownDrainsQueuedJobs` : tous les jobs soumis avant `Shutdown` sont bien traités avant
  qu'il ne rende la main.
- `TestPool_SubmitAfterShutdownReturnsFalse` : un `Submit` après `Shutdown` échoue sans paniquer.
- `TestPool_ShutdownRespectsContextDeadline` : si les workers ne finissent pas à temps, `Shutdown`
  remonte `context.DeadlineExceeded` plutôt que d'attendre indéfiniment.
