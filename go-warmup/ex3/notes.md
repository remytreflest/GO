# Ex3 — Structs, méthodes et defer

## Struct

```go
type Note struct {
    Title   string
    Content string
    Tags    []string
}
```

Les champs exportés (majuscule) sont accessibles hors package. Champs non exportés = privés.

## Receiver valeur vs pointeur

```go
func (n Note) Preview() string { ... }    // copie, ne modifie pas n
func (n *Note) AddTag(tag string) { ... } // modifie n via pointeur
```

**Pourquoi :** Si la méthode doit muter l'objet, il faut un receiver pointeur. Sinon le receiver valeur travaille sur une copie. Règle pratique : si une méthode du type est sur pointeur, toutes le sont.

## Constructeur par convention

```go
func NewNote(title, content string) *Note {
    return &Note{Title: title, Content: content}
}
```

Go n'a pas de constructeurs. La convention est une fonction `New<Type>`. Retourner `*Note` permet de chaîner des méthodes et évite les copies.

## `defer`

```go
f, err := os.Open(path)
if err != nil { return nil, err }
defer f.Close()   // s'exécute quand la fonction retourne, quelle que soit la raison
```

**Pourquoi :** Garantit la libération de ressources même si une erreur survient plus bas. Évite les fuites de fichiers ouverts. Les defers s'empilent (LIFO).

## Dédup dans un slice

```go
func (n *Note) AddTag(tag string) {
    for _, t := range n.Tags {
        if t == tag { return }
    }
    n.Tags = append(n.Tags, tag)
}
```

Pas de Set en Go. On itère pour vérifier l'absence avant d'ajouter.
