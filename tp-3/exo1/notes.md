# Exo1 — Première goroutine

## `go f()` lance une fonction en arrière-plan

```go
go afficherLettres()
afficherChiffres()
```

`afficherLettres()` s'exécute dans une nouvelle goroutine, en parallèle de
`afficherChiffres()` qui tourne dans la goroutine principale (celle de `main`).

## Pourquoi un `time.Sleep` final dans `main` ?

Le programme Go se termine dès que la fonction `main` retourne, **même si
d'autres goroutines sont encore en cours d'exécution**. Il n'y a aucune
attente implicite des goroutines lancées avec `go`.

**Question posée : que se passe-t-il si on retire le `time.Sleep` final ?**

`main` termine dès que `afficherChiffres()` a fini d'afficher ses 5 chiffres
(~250 ms). À ce moment, `afficherLettres()` n'a peut-être affiché que
quelques lettres, voire aucune : le process s'arrête brutalement et la
goroutine est tuée avant d'avoir terminé. Le résultat devient non
déterministe (on peut voir 0, 1, 2... lettres selon le scheduler).

C'est exactement le problème que corrige l'exo2 avec `sync.WaitGroup` : on
remplace une attente arbitraire (et fragile) par une synchronisation
explicite qui garantit que la goroutine a fini avant que `main` ne continue.
