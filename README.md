# Mira

Application de prise de notes en ligne de commande, écrite en Go.

## Commandes Go essentielles

| Commande | Description |
|---|---|
| `go run .` | Compile et exécute sans produire de binaire |
| `go build .` | Compile et produit un binaire (`mira.exe` sur Windows) |
| `go build -o nom.exe .` | Compile avec un nom de sortie personnalisé |
| `go fmt ./...` | Formate tout le code source |
| `go vet ./...` | Analyse statique — détecte les erreurs courantes |
| `go test ./...` | Lance tous les tests |
| `go mod tidy` | Nettoie et synchronise les dépendances |
| `go get <module>` | Ajoute une dépendance |

**Cross-compilation** (depuis Windows vers Linux) :
```bash
GOOS=linux GOARCH=amd64 go build -o mira .
```

## Utilisation

```bash
go run .
```

Commandes disponibles dans l'application :

- `ajouter` — Créer une nouvelle note
- `lire` — Afficher toutes les notes
- `quitter` — Quitter Mira
