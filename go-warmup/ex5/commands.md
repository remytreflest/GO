# Ex5 — Commandes

## Objectifs
- Écrire des tests unitaires avec le package `testing` (convention `TestXxx`).
- Distinguer `t.Fatal` (arrêt immédiat) de `t.Error` (continue).
- Tester les erreurs typées avec `errors.Is`.
- Intégrer `go fmt`, `go vet` et `go test` comme réflexes systématiques.

---

Depuis la racine du module (`mira/`) :

```bash
# Formater
go fmt ./go-warmup/ex5/...

# Analyse statique
go vet ./go-warmup/ex5/...

# Lancer les tests (verbose)
go test ./go-warmup/ex5/... -v

# Lancer un test précis
go test ./go-warmup/ex5/... -run TestSave_duplicate -v

# Lancer tous les tests du projet
go test ./... -v
```
