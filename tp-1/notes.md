# TP-1 — Mira CLI locale

## JSON Lines (JSONL) vs JSON array

```
{"id":"abc","title":"Go","content":"..."}   ← ligne 1
{"id":"def","title":"REST","content":"..."}  ← ligne 2
```

Pas de `[...]` ni de virgules entre objets. Chaque ligne est un JSON valide indépendant.

**Pourquoi :** Append = écriture d'une seule ligne, pas de ré-encodage du fichier entier. Lisible avec `grep`, `jq`. Standard pour les logs et datasets.

## `json.NewEncoder(f).Encode(n)` ajoute `\n` automatiquement

C'est ce qui crée le format JSONL : chaque `Encode` écrit l'objet + un saut de ligne.

## `bufio.Scanner` pour lire le JSONL ligne par ligne

```go
sc := bufio.NewScanner(f)
for sc.Scan() {
    json.Unmarshal(sc.Bytes(), &n)
}
```

`sc.Bytes()` retourne la ligne courante sans le `\n`. Plus efficace que `ioutil.ReadAll` sur de grands fichiers.

## `os.UserHomeDir()` — chemin home cross-platform

```go
home, _ := os.UserHomeDir()  // C:\Users\mrtre sur Windows, /home/mrtre sur Linux
dir := filepath.Join(home, ".mira")
```

**Pourquoi :** `~` ne fonctionne pas en Go (c'est une convention shell, pas une API OS).

## `os.MkdirAll` — créer un dossier récursivement

```go
os.MkdirAll(dir, 0755)  // crée ~/.mira/ si absent, sans erreur si déjà existant
```

Équivalent de `mkdir -p`. `0755` = rwxr-xr-x (propriétaire peut écrire, les autres lisent).

## `internal/` — visibilité restreinte

```
tp-1/internal/notes/    → importable uniquement depuis tp-1/
tp-1/internal/search/   → importable uniquement depuis tp-1/
```

Go enforce cette règle à la compilation. Empêche d'autres packages du module d'importer ces internals.

## Séparation `Store` (interface) / `JSONLStore` (implémentation)

`store.go` définit l'interface, `jsonl.go` l'implémente. `main.go` ne connaît que `notes.Store`.

**Pourquoi :** Permet de brancher une autre implémentation (SQLite, mémoire pour les tests) sans toucher `main.go`. C'est le pattern repository.

## `List()` — les 10 dernières par suffix de slice

```go
if len(notes) > 10 {
    notes = notes[len(notes)-10:]
}
```

Les notes sont ordonnées par insertion dans le fichier JSONL. Le suffix donne les plus récentes.

## Filtrage JSONL sans copie inutile

```go
filtered := make([]*Note, 0, len(notes))
for _, n := range notes {
    if n.ID == id { found = true; continue }
    filtered = append(filtered, n)
}
```

Pattern idiomatique pour supprimer un élément d'un slice sans modifier l'original.

## `flag.NewFlagSet` + args positionnels

```go
fs.Parse(os.Args[2:])
args := fs.Args()   // args non-flag restants
title, content = args[0], args[1]
```

Permet de mixer `mira add "titre" "contenu" -tags go` : les flags et les positionnels coexistent.
