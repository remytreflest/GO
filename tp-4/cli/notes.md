# TP-4 — Mira CLI (client HTTP)

Évolution de `tp-1/`. Les sections ci-dessous, reprises de `tp-1/notes.md`, restent valables pour
tout ce qui est inchangé (pattern repository, `flag.NewFlagSet`, `internal/`). Les nouveautés
propres au TP4 (bascule vers un client HTTP) sont en fin de fichier.

## JSON Lines (JSONL) vs JSON array

*(Section historique : tp-1 stockait les notes en JSONL local. tp-4/cli ne touche plus ce fichier —
voir "Le CLI passe par l'API" en fin de document.)*

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

## Séparation `Store` (interface) / implémentation

`store.go` définit l'interface `notes.Store`. `main.go` ne connaît que cette interface — jamais
l'implémentation concrète.

**Pourquoi :** permet de brancher une autre implémentation sans toucher `main.go`. C'est le pattern
repository, et c'est exactement ce qui a permis de remplacer `JSONLStore` (tp-1, fichier local) par
`apiclient.Client` (tp-4, HTTP) **sans changer une seule ligne de `store.go`** ni des signatures des
commandes CLI qui prennent un `notes.Store` en paramètre.

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

Permet de mixer flags et positionnels — **attention à l'ordre** : `flag.Parse` s'arrête au premier
argument qui n'est pas un flag. `mira add -tags go "titre" "contenu"` fonctionne (les flags sont
parsés avant que le premier positionnel soit rencontré) ; `mira add "titre" "contenu" -tags go` ne
fonctionne **pas** (`-tags` atterrit dans `fs.Args()` comme positionnel supplémentaire, jamais
reconnu comme flag). C'est un piège découvert en testant `tp-4/cli update <id> -content "..."` en
conditions réelles (voir plus bas) — les usages de `tp-4/cli` placent donc systématiquement les
flags en premier.

---

# Nouveautés TP-4 : le CLI passe par l'API

## `notes.Store`, même interface, nouvelle implémentation

`internal/apiclient.Client` implémente `notes.Store` (`Save`/`Get`/`Delete`/`List`/`All`) en
appelant l'API HTTP plutôt qu'un fichier local. `main.go`/`interactive.go` ne changent quasiment pas
— seule la construction du store change (`apiclient.New(apiURL)` au lieu de
`notes.NewJSONLStore()`).

## L'ID est désormais assigné par le serveur

`notes.New()` (côté client) pré-remplit un ID local jetable
(`fmt.Sprintf("%x", time.Now().UnixNano())`) — un vestige de tp-1, où ce Store écrivait directement
le fichier. Depuis l'API, c'est `core.NewID()` (côté serveur) qui fait foi. `apiclient.Client.Save`
réécrit donc `n.ID` (et `CreatedAt`/`Summary`/`Score`/`EnrichmentStatus`) avec la réponse du serveur
juste après le POST, pour que le code appelant (qui lit `n.ID` juste après `Save`) continue de
fonctionner sans modification. Conséquence : `notes.ErrDuplicate` (tp-1) est de facto inatteignable
depuis le CLI — le serveur ne génère jamais deux fois le même ID.

## `All()` : pagination transparente

`notes.Store.All()` doit renvoyer toutes les notes (utilisé historiquement par la recherche locale).
Contre une API paginée, `apiclient.Client.All()` boucle sur `GET /api/v1/notes?limit=100&offset=N`
jusqu'à ce qu'une page soit vide ou que `offset` atteigne `total`, avec un plafond de sécurité
(1000 pages) contre un `total` mal rapporté par le serveur qui bouclerait sinon indéfiniment.

## `search` ne passe plus par `All()` + filtrage local

tp-1 chargeait tout (`All()`) et filtrait côté client (`internal/search.Filter`, sous-chaîne naïve).
Ce package n'a **pas de successeur** dans tp-4/cli — choix délibéré : la recherche hybride
(full-text + similarité vectorielle) n'a de sens que côté serveur, car c'est lui qui détient les
embeddings produits par l'enrichissement. `apiclient.Client.Search(query)` appelle directement
`GET /api/v1/search?q=...` ; ce n'est pas ajouté à l'interface `notes.Store` (les autres
implémentations n'auraient aucun moyen raisonnable de la satisfaire), donc `cmdSearch`/
`interactiveSearch` prennent un `*apiclient.Client` concret plutôt que `notes.Store`.

## Nouvelle commande `update` (absente de tp-1)

tp-1 n'avait pas de commande de mise à jour. `tp-4/cli update` a été ajoutée spécifiquement pour
pouvoir déclencher, depuis le CLI, le chemin PATCH → ré-enrichissement côté API (`enrichment_status`
repasse à `pending` puis `done`) — sans elle, il n'y aurait aucun moyen d'observer ce
comportement autrement qu'en curl.

## Vérification au démarrage (`Ping`)

`main.go` appelle `client.Ping()` avant de traiter la moindre commande. Sans ça, la première requête
réelle (ex. `add`) échouerait avec un message de timeout `net/http` peu clair si l'API n'est pas
lancée. `Ping` échoue vite avec un message explicite ("cannot reach Mira API at ...").
