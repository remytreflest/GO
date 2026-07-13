# Ex1 — Variables, types et boucles

## Zero values

Toute variable déclarée sans valeur initiale a une valeur par défaut utile, pas `null`.

```go
var s string  // ""
var i int     // 0
var b bool    // false
```

**Pourquoi c'est important :** Pas de null pointer panic sur les types de base. Comportement prévisible.

## `:=` vs `var`

```go
x := 0        // inférence de type, uniquement dans une fonction
var y int     // explicite, fonctionne aussi au niveau package
```

Les deux créent un `int` valant `0`. `:=` est le raccourci usuel dans les corps de fonctions.

## `os.Args`

```go
os.Args        // []string, os.Args[0] = chemin du binaire
os.Args[1:]    // les arguments réels
```

**Pourquoi :** Point d'entrée standard pour les programmes CLI sans librairie externe.

## `os.Exit(1)`

Termine immédiatement le processus avec un code non-zero. Les `defer` **ne s'exécutent pas**.

**Pourquoi :** Convention Unix — code 0 = succès, non-zero = erreur. Les scripts shell s'appuient dessus.

## `for range` sur un slice

```go
for i, word := range words {
    // i = index, word = copie de l'élément
}
for _, word := range words {
    // _ ignore l'index
}
```

## `const`

```go
const MaxDisplay = 10
```

Évalué à la compilation. Ne peut pas être modifié. Préférer aux magic numbers.
