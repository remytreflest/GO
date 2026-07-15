# Exo6 — Commandes

```bash
# Version avec la race condition
go run ./tp-3/exo6/race

# Avec le détecteur de race (nécessite cgo + un compilateur C installé)
go run -race ./tp-3/exo6/race

# Version corrigée avec sync.Mutex
go run ./tp-3/exo6/fixed
```

## Installer gcc pour `-race` (Windows)

Installé ici via Chocolatey :

```powershell
choco install mingw -y
```

gcc se trouve dans `C:\ProgramData\mingw64\mingw64\bin`, déjà ajouté au PATH
machine par le package. **Redémarrer le terminal / VS Code** après
l'installation : les terminaux déjà ouverts gardent l'ancien PATH en
mémoire tant qu'ils ne sont pas relancés. Vérifier ensuite avec :

```powershell
gcc --version
go run -race ./tp-3/exo6/race
```
