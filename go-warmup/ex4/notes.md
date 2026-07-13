# Ex4 — Interfaces et erreurs typées

## Interface implicite

```go
type NoteStore interface {
    Save(n *Note) error
    Get(id string) (*Note, error)
    All() []*Note
}
```

`MemoryStore` implémente `NoteStore` si elle a ces 3 méthodes — sans déclarer `implements`. Le compilateur vérifie à l'usage.

**Pourquoi :** Découplage. Le code qui consomme `NoteStore` ne connaît pas `MemoryStore`. On peut substituer une implémentation disque/DB sans changer le code appelant.

## Erreurs sentinelles

```go
var ErrDuplicate = errors.New("note already exists")
var ErrNotFound  = errors.New("note not found")
```

Des variables globales comparables par identité. Jamais recréer avec `errors.New` à l'intérieur d'une fonction — ce serait une nouvelle erreur à chaque appel.

**Pourquoi :** Permet à l'appelant de tester le type d'erreur sans parser le message.

## `errors.Is`

```go
if errors.Is(err, ErrNotFound) { ... }
```

Préférer `errors.Is` à `err == ErrNotFound`. `errors.Is` traverse la chaîne de wrapping (`%w`).

## `sync.Mutex`

```go
type MemoryStore struct {
    mu    sync.Mutex
    notes map[string]*Note
}

func (s *MemoryStore) Save(n *Note) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    // ...
}
```

**Pourquoi :** Une map lue/écrite depuis plusieurs goroutines sans mutex = race condition = comportement indéfini. Le `defer Unlock` garantit le déverrouillage même en cas de `return` anticipé.

## Mutation via pointeur et interface

Si `MemoryStore` a des receivers pointeur (`*MemoryStore`), seul `*MemoryStore` satisfait l'interface, pas `MemoryStore`. Toujours passer/stocker l'adresse.
