# Exo3 — Somme parallèle avec channels

## Un channel comme point de rendez-vous

```go
resultat := make(chan int) // channel non bufferisé
go sommePartielle(nums[debut:fin], resultat)
...
somme += <-resultat
```

Contrairement à `WaitGroup` (qui ne fait que signaler "j'ai fini"), un
channel permet de **transporter une valeur** entre goroutines. Ici, chaque
goroutine calcule une somme partielle sur un quart du slice puis l'envoie
sur `resultat`.

## `<-resultat` bloque jusqu'à réception

La boucle `for i := 0; i < 4; i++ { somme += <-resultat }` reçoit exactement
4 valeurs, une par goroutine lancée. L'ordre de réception ne correspond pas
forcément à l'ordre de lancement des goroutines (la première goroutine
finie est la première reçue), mais peu importe ici puisqu'on additionne.

## Découpage d'un slice en morceaux

```go
nums[debut:fin]
```

Un sous-slice ne copie pas les données : il partage le même tableau
sous-jacent que `nums`. C'est sûr ici car chaque goroutine lit une plage
disjointe et personne n'écrit dans `nums` en parallèle.
