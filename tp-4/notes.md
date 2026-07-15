# TP-4 — PostgreSQL, enrichissement automatique, recherche hybride

Vue d'ensemble ; le détail (notions clés) de chaque partie est dans `api/notes.md` et `cli/notes.md`.

## Structure

```
tp-4/
├── docker-compose.yml   # Postgres (image pgvector/pgvector:pg16), port hôte 5433
├── .env.example         # DATABASE_URL, PORT, ENRICHMENT_*, MIRA_API_URL
├── api/                 # évolution de tp-2 : Postgres + enrichissement async + recherche hybride
└── cli/                 # évolution de tp-1 : client HTTP au lieu du JSONL local
```

`tp-1/` et `tp-2/` restent des snapshots figés de leur TP respectif — `tp-4/api` et `tp-4/cli` en
sont des copies évoluées, pas des modifications en place.

## Trois évolutions majeures

1. **PostgreSQL** (`api/internal/store/postgres`) : schéma + migrations (`api/migrations/`,
   golang-migrate + `embed.FS`, appliquées automatiquement au démarrage), repository via **pgx**
   (`pgxpool`), transaction sur création de note + tags.

2. **Enrichissement automatique** (`api/internal/enrichment` + `api/internal/embedding`) : à chaque
   POST/PATCH, un job (tags/résumé/score/embedding — calculés **localement, sans API externe**) est
   publié dans un channel interne ; un pool de workers borné le traite avec un timeout `context` par
   tâche ; le résultat est écrit en base ; la note porte un `enrichment_status`
   (`pending`/`done`/`failed`). L'écriture de la note reste synchrone et rapide — l'enrichissement
   suit en asynchrone, sans jamais bloquer la requête HTTP (`Submit` non-bloquant).

3. **Recherche hybride** (`api/internal/store/postgres/search.go`) : full-text (`tsvector` + index
   GIN) combiné à la similarité vectorielle (pgvector, index HNSW) sur les embeddings produits par
   l'enrichissement.

4. **Bonus/conséquence** : la CLI (`cli/`) passe désormais par l'API (client HTTP,
   `cli/internal/apiclient`) et non plus par le fichier JSONL local — c'est le seul moyen de
   garantir que toute note créée ou modifiée déclenche l'enrichissement automatique.

## Décisions de conception notables

- **Enrichissement simulé, 100% local** : pas d'appel à un service d'IA externe, pas de clé API à
  gérer. Tags = mots-clés fréquents, résumé = troncature intelligente, score = heuristique sur le
  contenu, embedding = vecteur pseudo-sémantique déterministe (hachage de sac-de-mots + norme L2).
  Cohérent avec l'esprit offline/testable des TP précédents ; voir `api/notes.md` pour le détail et
  les limites (pas de vraie notion de synonyme/sens).
- **Nouveau dossier `tp-4/`, pas de modification en place** : `tp-1/`/`tp-2/` restent des
  références pédagogiques figées.
- **Pool de workers + timeout `context`** : reprend directement les patterns de `tp-3/exo4` (worker
  pool) et `tp-3/bonus` (`context.WithTimeout`).
