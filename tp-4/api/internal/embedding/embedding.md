# `internal/embedding` — enrichissement simulé, 100% local

Ce package calcule les 4 informations qu'un note obtient lors de son enrichissement asynchrone
(`tags`, `summary`, `score`, `embedding`). Aucune de ces fonctions ne fait d'appel réseau ou d'API
externe (pas d'OpenAI, pas de modèle ML) : tout est déterministe, pur, et calculable en local — un
choix volontaire pour rester offline/testable en CI, comme le reste des TP.

Ordre de lecture logique : `tokenize.go` (la brique commune) → `embedding.go` (le cœur du package) →
`tags.go` → `summary.go` → `score.go`.

## `tokenize.go` — la brique partagée

```go
func tokenize(text string) []string
```

Découpe un texte en une liste de mots : tout est mis en minuscule, puis on coupe sur tout caractère
qui n'est ni une lettre ASCII, ni un chiffre, ni un caractère non-ASCII (`r > 127`, pour ne pas
casser les accents comme "é"). `"Goroutines, channels!"` → `["goroutines", "channels"]`.

C'est la seule fonction non exportée (minuscule) du package : elle n'a de sens que comme brique
interne, réutilisée par `Embed`, `ExtractTags` et `Score`.

## `embedding.go` — `Embed`, le vecteur "pseudo-sémantique"

```go
const Dimension = 64
func Embed(text string) []float32
```

Retourne un vecteur de 64 flottants (`float32`), normalisé (longueur 1), qui représente le texte.
C'est ce vecteur qui est ensuite stocké dans pgvector et comparé par similarité cosinus lors d'une
recherche (voir `search.go` dans `internal/store/postgres`).

Comment il est construit :

1. `tokenize(text)` découpe le texte en mots. Si le texte est vide, on utilise un mot sentinelle
   (`__empty__`) plutôt que de laisser la liste de tokens vide — sinon le vecteur produit serait
   nul, et la similarité cosinus contre le vecteur nul n'est pas mathématiquement définie.
2. **Feature hashing** : chaque mot est haché (FNV-32a) puis réduit modulo `Dimension` pour choisir
   une des 64 "cases" du vecteur ; on incrémente cette case à chaque occurrence du mot. C'est un
   "sac de mots" (bag-of-words) compressé dans une taille fixe, sans avoir besoin de maintenir un
   vocabulaire explicite qui grandirait indéfiniment.
3. **Normalisation L2** : le vecteur est divisé par sa norme pour que sa longueur vaille 1. Ça rend
   la similarité cosinus comparable entre un texte court et un texte long.

**Ce que ce n'est pas** : un vrai embedding sémantique (type modèle de langage). Deux textes qui
partagent des mots se retrouvent proches dans l'espace vectoriel, mais il n'y a aucune notion de
synonyme ou de sens — `"goroutine"` et `"goroutines"` sont deux tokens différents, donc deux cases
différentes, donc pas forcément proches.

## `tags.go` — `ExtractTags`, les mots-clés dominants

```go
func ExtractTags(text string, max int) []string
```

Retourne jusqu'à `max` mots les plus fréquents du texte, en excluant :
- les mots de 2 caractères ou moins,
- une petite liste de mots vides (`stopwords`) FR/EN codée en dur (`"le"`, `"the"`, `"pour"`,
  `"with"`, ...).

Le tri est fait par fréquence décroissante, puis par ordre alphabétique en cas d'égalité. Ce
deuxième critère n'est pas cosmétique : l'itération sur une map Go est volontairement aléatoire par
le runtime, donc sans ce tri explicite, deux appels avec le même texte pourraient renvoyer les tags
dans un ordre différent — mauvais pour les tests et pour la reproductibilité.

## `summary.go` — `Summarize`, un résumé sans IA

```go
func Summarize(text string, maxLen int) string
```

Règle simple en deux temps :
1. Si le texte contient un signe de ponctuation de fin de phrase (`.`, `!`, `?`) avant `maxLen`, on
   renvoie la première phrase telle quelle.
2. Sinon, on tronque à `maxLen` caractères, en reculant jusqu'au dernier espace pour ne pas couper un
   mot en deux, et on ajoute une ellipse (`…`).

## `score.go` — `Score`, une heuristique de "richesse"

```go
func Score(title, content string) float64
```

Renvoie une note entre 0 et 1, combinant à parts égales :
- la **longueur** du contenu (plafonnée à 1000 caractères → score de longueur max),
- la **diversité du vocabulaire** (nombre de mots uniques / nombre total de mots).

Le commentaire du code est explicite : ce n'est **pas** un jugement de qualité, juste un signal
illustratif suffisant pour démontrer que le pipeline d'enrichissement sait écrire une valeur
calculée dans la note.

## Comment ces 4 fonctions s'articulent dans le pipeline

Elles sont appelées ensemble, une seule fois par note, dans le worker d'enrichissement
(`internal/enrichment/pool.go`, fonction `process`) :

```go
text := job.Title + " " + job.Content
result := core.EnrichmentResult{
    Tags:      embedding.ExtractTags(text, maxTags),
    Summary:   embedding.Summarize(job.Content, summaryMaxLen),
    Score:     embedding.Score(job.Title, job.Content),
    Embedding: embedding.Embed(text),
    Status:    core.EnrichmentDone,
}
```

Ce calcul est pur et instantané (pas d'I/O) ; c'est volontairement séparé de l'écriture en base
(`SaveEnrichment`), qui elle est protégée par un timeout — voir `tp-4/api/notes.md`, section "Pool
de workers borné", pour la suite du pipeline (écriture asynchrone, gestion des erreurs).

Pour la partie stockage/recherche du vecteur produit par `Embed` (pgvector, index HNSW, requête
hybride texte+vecteur), voir `tp-4/api/notes.md`, section "Recherche hybride".
