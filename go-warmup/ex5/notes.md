# Ex5 — Tests unitaires

## Convention de nommage

```go
// fichier : store_test.go  (même package ou package_test)
func TestSave_valid(t *testing.T)      { ... }
func TestSave_emptyTitle(t *testing.T) { ... }
```

Préfixe `Test` obligatoire. Le compilateur exclut les fichiers `_test.go` du binaire final.

## `t.Fatal` vs `t.Error`

```go
t.Fatal("message")  // stoppe le test immédiatement (comme t.FailNow)
t.Error("message")  // marque le test échoué mais continue
```

Utiliser `t.Fatal` quand la suite du test n'a pas de sens si l'assertion échoue.

## Tester les erreurs

```go
if !errors.Is(err, ErrDuplicate) {
    t.Fatalf("want ErrDuplicate, got %v", err)
}
```

Toujours `errors.Is`, pas `==`, pour être compatible avec le wrapping futur.

## Commandes essentielles

```bash
go fmt ./...        # formate tout le code — obligatoire avant commit
go vet ./...        # détecte les erreurs statiques courantes (Printf mal formé, etc.)
go test ./... -v    # lance tous les tests, -v = affiche chaque cas
```

**Pourquoi `go fmt` est non négociable :** Le style est imposé par l'outil. Pas de débat de style, diff lisible, revues de code focalisées sur la logique.

## Structure d'un test table-driven (bonus)

```go
tests := []struct {
    name    string
    input   string
    wantErr error
}{
    {"valid", "My note", nil},
    {"empty title", "", ErrValidation},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ...
    })
}
```

Évite la duplication quand on teste la même fonction avec des variantes d'input.
