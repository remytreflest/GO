# TP-1 — Commandes

## Objectifs
- Structurer un projet Go avec des packages internes (`internal/`).
- Stocker des données en **JSON Lines** dans `~/.mira/notes.jsonl`.
- Implémenter le pattern repository : interface `Store` + implémentation `JSONLStore`.
- Séparer la logique de recherche dans un package dédié (`internal/search`).
- Garder le mode interactif et les commandes `get` / `delete` de l'ex6-bonus.

---

## Structure

```
tp-1/
├── main.go               # dispatch CLI
├── interactive.go        # mode interactif (sans arguments)
└── internal/
    ├── notes/
    │   ├── note.go       # struct Note + constructeur
    │   ├── store.go      # interface Store + erreurs sentinelles
    │   └── jsonl.go      # JSONLStore : lecture/écriture ~/.mira/notes.jsonl
    └── search/
        └── search.go     # Filter(all, query) — recherche texte naïve
```

---

## Build

Depuis la racine du module (`mira/`) :

```bash
go build -o tp-1/mira.exe ./tp-1/
```

Lancer depuis n'importe où — les données sont toujours dans `~/.mira/notes.jsonl`.

---

## Commandes CLI

**Ajouter une note :**

```bash
.\tp-1\mira.exe add "Go interfaces" "Les interfaces sont implicites en Go"
.\tp-1\mira.exe add "REST API" "GET POST PUT DELETE" -tags api,rest
```

**Lister les 10 dernières notes :**

```bash
.\tp-1\mira.exe list
```

**Rechercher dans titres et contenus :**

```bash
.\tp-1\mira.exe search go
.\tp-1\mira.exe search "JSON Lines"
```

**Lire une note complète :**

```bash
.\tp-1\mira.exe get <id>
```

**Supprimer une note :**

```bash
.\tp-1\mira.exe delete <id>
```

---

## Mode interactif (sans arguments)

```bash
.\tp-1\mira.exe
```

```
=== Mira — mode interactif ===
> ajouter | lire | supprimer | lister | rechercher | quitter
:
```

Les champs sont demandés un par un. La persistance est automatique après chaque mutation.

---

## Données

Les notes sont stockées dans `%USERPROFILE%\.mira\notes.jsonl` (Windows) / `~/.mira/notes.jsonl` (Linux/Mac).  
Format : 1 objet JSON par ligne.
