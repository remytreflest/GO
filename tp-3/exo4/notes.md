# Exo4 — Worker pool

## Pattern worker pool

- `jobs` : channel bufferisé où `main` dépose le travail à faire.
- N goroutines `worker` lisent en boucle dans `jobs` (`for j := range jobs`)
  et écrivent leur résultat dans `resultats`.
- `main` envoie tous les jobs, **ferme `jobs`** (`close(jobs)`), attend la
  fin de tous les workers (`wg.Wait()`), puis **ferme `resultats`**.

## `close(jobs)` : comment les workers savent qu'il n'y a plus de travail

`for j := range jobs` boucle jusqu'à ce que le channel soit à la fois vide
**et** fermé. Fermer `jobs` après l'envoi de tous les jobs est ce qui permet
aux boucles `range` des workers de se terminer proprement, sans qu'on ait
besoin d'un signal explicite supplémentaire.

## Pourquoi fermer `resultats` seulement après `wg.Wait()` ?

Si on fermait `resultats` avant que tous les workers aient fini d'écrire
dedans, un worker qui tente encore un envoi sur un channel fermé provoque un
`panic: send on closed channel`. Il faut donc garantir que plus personne
n'écrit dans `resultats` avant de le fermer — d'où l'attente via
`wg.Wait()`.

## Question : pourquoi l'ordre des résultats n'est-il pas garanti ?

Réponse (aussi en commentaire dans `main.go`) : les 4 workers tournent en
parallèle et piochent dans `jobs` dès qu'ils sont libres. Le *scheduler* Go
décide de l'ordre d'exécution des goroutines selon des facteurs qu'on ne
contrôle pas (temps CPU disponible, préemption, etc.), pas selon l'ordre
dans lequel les jobs ont été envoyés. Deux exécutions successives peuvent
donc produire les résultats dans un ordre différent.
