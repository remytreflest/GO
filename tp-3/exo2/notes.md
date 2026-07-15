# Exo2 — Synchronisation propre avec WaitGroup

## `sync.WaitGroup` remplace un `time.Sleep` arbitraire

Un `time.Sleep` pour "attendre que les goroutines finissent" est fragile :
la durée est devinée, pas garantie. Trop courte, le programme coupe des
goroutines en cours ; trop longue, le programme est ralenti pour rien.

`sync.WaitGroup` est un compteur atomique thread-safe :

- `wg.Add(n)` : incrémente le compteur de `n` (nombre de goroutines à
  attendre). À faire **avant** de lancer les goroutines, dans la goroutine
  appelante — pas à l'intérieur de la goroutine elle-même (sinon `main`
  pourrait appeler `Wait()` avant que le `Add` n'ait eu lieu).
- `wg.Done()` : décrémente le compteur de 1. Généralement appelé via
  `defer wg.Done()` en première ligne de la goroutine, pour garantir
  l'appel même en cas de panique.
- `wg.Wait()` : bloque jusqu'à ce que le compteur retombe à 0.

## Passer `*sync.WaitGroup` par pointeur

`WaitGroup` ne doit jamais être copié après son premier usage (il contient
un compteur interne). On le passe donc toujours par pointeur aux fonctions
qui appellent `Done()`.
