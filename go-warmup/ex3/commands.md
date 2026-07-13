# Ex3 — Commandes

## Objectifs
- Définir une struct avec méthodes sur pointeur (`Preview`, `AddTag`).
- Écrire un constructeur `NewNote` par convention Go.
- Utiliser `defer f.Close()` pour garantir la libération de ressources.
- Lire/écrire du JSON avec `encoding/json`.
- Filtrer une slice de structs sur un champ (tag `"go"`).

---

Depuis la racine du module (`mira/`) :

```bash
go run ./go-warmup/ex3/
```

Le programme crée `go-warmup/ex3/notes.json` puis le relit — pas d'arguments nécessaires.

Build + exécution du binaire :

```bash
go build -o go-warmup/ex3/ex3.exe ./go-warmup/ex3/
./go-warmup/ex3/ex3.exe
```
