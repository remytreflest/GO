# TP-3 — Concurrence et goroutines en Go

Vue d'ensemble ; le détail (notions clés) de chaque exercice est dans
`exoN/notes.md` (et `bonus/notes.md`).

## Structure

```
tp-3/
├── exo1/   # goroutine simple + time.Sleep
├── exo2/   # sync.WaitGroup
├── exo3/   # somme parallèle via channels
├── exo4/   # worker pool (jobs/résultats + WaitGroup)
├── exo5/   # select + time.After (timeout)
├── exo6/
│   ├── race/   # code avec race condition (tel que fourni)
│   └── fixed/  # même code corrigé avec sync.Mutex
└── bonus/  # context.WithTimeout appliqué au worker pool
```

Tous les exercices font partie du module Go racine `mira` (pas de `go.mod`
séparé) : chaque dossier est un `package main` indépendant, exécutable via
`go run ./tp-3/<dossier>`.

## Fil conducteur

1. **exo1/exo2** : deux goroutines qui affichent en parallèle, d'abord
   synchronisées avec un `time.Sleep` arbitraire (fragile), puis avec un
   `sync.WaitGroup` (fiable).
2. **exo3** : les channels servent à transporter des valeurs (pas
   seulement à signaler) — ici des sommes partielles calculées en
   parallèle.
3. **exo4/exo5** : pattern worker pool complet, puis ajout d'un timeout
   côté consommateur avec `select`/`time.After`.
4. **exo6** : une race condition classique (`compteur++` non protégé),
   observée puis corrigée avec `sync.Mutex`.
5. **bonus** : `context.WithTimeout` propage l'annulation jusque dans les
   workers, contrairement au `select` de l'exo5 qui n'arrêtait que
   l'attente côté `main`.
