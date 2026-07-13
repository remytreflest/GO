# Ex1 — Commandes

## Objectifs
- Déclarer une constante et des variables, comprendre les zero values.
- Lire les arguments CLI via `os.Args`.
- Itérer avec `for range` et filtrer par longueur de chaîne.
- Retourner un code d'erreur avec `os.Exit(1)` si aucun argument n'est fourni.

---

Depuis la racine du module (`mira/`) :

```bash
go run ./go-warmup/ex1/ hello world foo bar longer
```

Sans arguments (doit retourner exit code 1) :

```bash
go run ./go-warmup/ex1/
```

Build + exécution du binaire :

```bash
go build -o go-warmup/ex1/ex1.exe ./go-warmup/ex1/
./go-warmup/ex1/ex1.exe hello world foo bar longer
```
