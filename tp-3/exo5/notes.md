# Exo5 — `select` et timeout

## `select` avec `time.After`

```go
select {
case r, ok := <-resultats:
    ...
case <-time.After(500 * time.Millisecond):
    fmt.Println("timeout sur un résultat")
}
```

`time.After(d)` renvoie un channel qui reçoit une valeur unique après la
durée `d`. Placé dans un `select` aux côtés d'un channel de résultat, il
permet d'abandonner l'attente si aucune valeur n'arrive à temps, sans
bloquer indéfiniment.

**Piège** : chaque itération de la boucle `for { select { ... } }` recrée un
nouveau timer de 500 ms. Le timeout est donc "glissant" : tant qu'aucun
résultat n'arrive, on imprime `"timeout sur un résultat"` toutes les 500 ms,
en boucle, jusqu'à ce que le résultat lent finisse par arriver.

## Le worker lent (`id == 1`) simule une latence

Un seul worker sur 4 dort 2 secondes avant de renvoyer chaque résultat.
Comme `resultats` est un channel **bufferisé** (capacité `nbJobs`), les
workers rapides peuvent y écrire sans attendre que `main` lise — leurs
résultats s'accumulent dans le buffer et sont consommés quasi instantanément
par la boucle `select`. Ce n'est qu'une fois le buffer vidé, en attendant le
résultat du worker lent, que le `select` bloque réellement et que le
`case <-time.After(...)` peut se déclencher.

## Sortie observée (un run typique)

```
résultat : 4
résultat : 9
... (17 autres résultats rapides)
timeout sur un résultat
timeout sur un résultat
timeout sur un résultat
résultat : 1
```

Durée totale ≈ 2,4 s (le temps que le worker 1 finisse son unique job avant
la fermeture de `jobs`). Le nombre de lignes `"timeout sur un résultat"`
varie d'une exécution à l'autre selon le timing exact du scheduler — il peut
même arriver, plus rarement, qu'aucun timeout ne s'affiche si le résultat
lent arrive juste avant l'expiration du timer en cours.
