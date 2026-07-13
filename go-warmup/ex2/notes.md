# Ex2 — Slices et maps

## Map : déclaration et utilisation

```go
counts := make(map[string]int)
counts["go"]++       // si "go" n'existe pas, part de 0 (zero value)
v, ok := counts["x"] // ok = false si absent
```

**Pourquoi :** La zero value des maps est `nil`, mais `make` initialise. Lire une clé absente ne panique pas — retourne la zero value du type valeur.

## Itération sur une map : ordre non garanti

```go
for key, val := range counts {
    // ordre aléatoire à chaque exécution
}
```

**Pourquoi :** La randomisation est intentionnelle depuis Go 1.0 pour éviter de dépendre d'un ordre non défini. Toujours trier si l'ordre compte.

## Trier une map par valeur : pattern slice de structs

```go
type tagCount struct {
    tag   string
    count int
}

pairs := make([]tagCount, 0, len(counts))
for k, v := range counts {
    pairs = append(pairs, tagCount{k, v})
}
sort.Slice(pairs, func(i, j int) bool {
    return pairs[i].count > pairs[j].count  // décroissant
})
```

**Pourquoi :** On ne peut pas trier une map directement. Il faut la convertir en slice de structs, puis `sort.Slice` avec un comparateur.

## `sort.Slice`

Prend un slice et une fonction `less(i, j int) bool`. Tri in-place, instable.

Pour un tri stable (préserve l'ordre des égaux) : `sort.SliceStable`.
