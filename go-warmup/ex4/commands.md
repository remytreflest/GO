# Ex4 — Commandes

## Objectifs
- Définir une interface `NoteStore` et l'implémenter implicitement avec `MemoryStore`.
- Créer des erreurs sentinelles (`ErrDuplicate`, `ErrNotFound`, `ErrValidation`).
- Comparer les erreurs avec `errors.Is` plutôt que `==`.
- Protéger une map partagée avec `sync.Mutex` + `defer Unlock`.

---

Depuis la racine du module (`mira/`) :

```bash
go run ./go-warmup/ex4/
```

Le programme démontre Save / Get / ErrDuplicate / ErrValidation / ErrNotFound — pas d'arguments.

Build + exécution du binaire :

```bash
go build -o go-warmup/ex4/ex4.exe ./go-warmup/ex4/
./go-warmup/ex4/ex4.exe
```
