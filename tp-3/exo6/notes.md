# Exo6 — Trouver et corriger une race condition

## Où est la race ?

```go
go func() {
    defer wg.Done()
    compteur++
}()
```

`compteur++` n'est **pas atomique** : c'est en réalité trois opérations
(lire `compteur`, ajouter 1, réécrire `compteur`). Quand 1000 goroutines
font ça en parallèle sur la même variable sans protection, deux goroutines
peuvent lire la même valeur avant que l'une des deux n'ait eu le temps
d'écrire son incrément : un des deux incréments est alors perdu.

## Réponses aux questions

**1. Résultat sans correction (plusieurs exécutions), dossier `race/`**

```
Compteur final : 985
Compteur final : 993
Compteur final : 992
```

Le résultat est systématiquement **inférieur ou égal à 1000** et varie
d'une exécution à l'autre — la signature typique d'une race condition sur
un compteur partagé (incréments perdus).

**2. `go run -race main.go`**

`-race` nécessite cgo (`CGO_ENABLED=1`) et donc un compilateur C, installé
ici via Chocolatey (`choco install mingw`, gcc extrait dans
`C:\ProgramData\mingw64\mingw64\bin`).

Rapport obtenu (`go run -race ./tp-3/exo6/race`) :

```
==================
WARNING: DATA RACE
Read at 0x00c000110078 by goroutine 10:
  main.main.func1()
      .../tp-3/exo6/race/main.go:16 +0x7b

Previous write at 0x00c000110078 by goroutine 8:
  main.main.func1()
      .../tp-3/exo6/race/main.go:16 +0x8d

Goroutine 10 (running) created at:
  main.main()
      .../tp-3/exo6/race/main.go:14 +0x78

Goroutine 8 (finished) created at:
  main.main()
      .../tp-3/exo6/race/main.go:14 +0x78
==================
Compteur final : 974
Found 2 data race(s)
exit status 66
```

Le rapport montre exactement le mécanisme de la race : une goroutine lit
`compteur` (`Read at ...`) pendant qu'une autre y écrit encore
(`Previous write at ...`), à la même ligne 16 (`compteur++`). Chaque bloc
donne aussi l'endroit où les deux goroutines en conflit ont été lancées
(`Goroutine N created at ...`, ligne 14 = `go func() { ... }()`). Le
détecteur sort avec le code 66 dès qu'il trouve au moins une race.

**3. Après correction avec `sync.Mutex`, dossier `fixed/`**

```
Compteur final : 1000
Compteur final : 1000
Compteur final : 1000
```

Stable et correct : `mu.Lock()` / `mu.Unlock()` autour de `compteur++`
garantit qu'une seule goroutine à la fois lit-modifie-écrit `compteur`,
éliminant la race. Confirmé aussi par `go run -race ./tp-3/exo6/fixed` :
aucune race détectée, juste `Compteur final : 1000`.
