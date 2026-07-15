# Bonus — `context.Context`

## `context.WithTimeout` vs `select` + `time.After` côté `main`

Dans l'exo5, seul `main` abandonnait l'attente d'un résultat — les workers,
eux, continuaient de tourner en arrière-plan même après le timeout affiché
(le `select` de `main` ne fait qu'arrêter d'*attendre*, il n'arrête rien).

`context.WithTimeout` propage un signal d'annulation **jusque dans les
workers** : chaque worker reçoit le même `ctx` et vérifie `<-ctx.Done()`
à chaque étape (avant de "traiter" le job, avant d'envoyer le résultat).
Dès que le délai expire, `ctx.Done()` se ferme pour tout le monde
simultanément, et les workers encore actifs s'arrêtent proprement au lieu
de continuer à travailler pour rien.

## Pourquoi deux `select` imbriqués dans `worker` ?

```go
select {
case <-time.After(delai):
case <-ctx.Done():
    return
}
select {
case resultats <- j * j:
case <-ctx.Done():
    return
}
```

Il faut pouvoir annuler à **deux moments distincts** : pendant le
traitement simulé (`time.Sleep`/`time.After`) et pendant l'envoi du résultat
(qui pourrait bloquer si `main` a déjà cessé de lire `resultats`). Sans le
second `select`, un worker annulé pourrait rester bloqué indéfiniment sur
`resultats <- ...` si personne ne le lit plus.

## `defer cancel()`

`context.WithTimeout` retourne aussi une fonction `cancel`. Il faut
toujours l'appeler (via `defer`) même si le timeout se déclenche tout seul,
pour libérer immédiatement les ressources internes associées au contexte
plutôt que d'attendre le garbage collector.

## Sortie observée

```
... (résultats rapides des workers 2, 3, 4)
worker 1 annulé (job 1 abandonné)
délai global d'1s dépassé, arrêt du traitement
```

Durée totale ≈ 1,9 s : le worker lent (2 s de traitement) est bien coupé
par le timeout de 1 s au lieu d'aller jusqu'au bout comme dans l'exo5.
