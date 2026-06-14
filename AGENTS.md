# AGENTS.md

Instructions pour les agents et contributeurs travaillant sur ce dépôt.

## Présentation

`hfr-mcp` fournit deux binaires Go pour [forum.hardware.fr](https://forum.hardware.fr) :

- `hfr-mcp` — serveur [MCP](https://modelcontextprotocol.io/) (transport stdio)
- `hfr` — CLI standalone

Fonctions : lire des topics, poster/éditer des réponses, citer (mono/multi), envoyer des MP, lister catégories et topics, créer un topic.

## Build

```bash
go build ./cmd/hfr-mcp/   # serveur MCP
go build ./cmd/hfr/        # CLI
go build ./...             # tout
```

Go 1.25+ (voir `go.mod`). Dépendances clés : `modelcontextprotocol/go-sdk`, `PuerkitoBio/goquery`.

## Vérification avant push

```bash
go vet ./...
golangci-lint run        # golangci-lint v2 (la CI épingle v2.11.4)
go build ./...
go test ./...            # pas encore de tests ; en ajouter avec toute nouvelle logique
```

La CI (`.github/workflows/ci.yml`) lance `go vet` + `golangci-lint` v2 + le build des deux binaires sur chaque push et PR. Reproduire localement avant de pousser.

## Structure

```
cmd/hfr/main.go            CLI : sous-commandes, parsing des args, --auth
cmd/hfr-mcp/main.go        Serveur MCP : login lazy, transport stdio
internal/hfr/client.go     Client HTTP, login, hash_check (anti-CSRF), cookie jar
internal/hfr/reader.go     Lecture de topics, mode print, batch concurrent
internal/hfr/parser.go     Parsing HTML (goquery), nettoyage du contenu
internal/hfr/post.go       Reply + Edit (détection du first post)
internal/hfr/mp.go         Messages privés
internal/hfr/categories.go Catégories et sous-catégories
internal/hfr/models.go     Post, Topic, TopicListItem, EditInfo
internal/hfr/errors.go     Types d'erreurs HFR
internal/hfr/version.go    Constante de version
internal/config/config.go  Fichier de config + env vars + vérif des permissions
internal/mcp/tools.go      Déclaration des outils MCP + handlers
internal/mcp/helpers.go    Formatage des résultats
```

Les structs d'entrée MCP portent des tags `jsonschema` : le SDK en dérive le schéma JSON automatiquement. **Pour ajouter un outil** : définir la struct d'entrée et le handler dans `internal/mcp/tools.go`, puis l'enregistrer via `mcp.AddTool` dans `RegisterTools`.

## Versions et releases

- Branches : `dev` = développement, `master` = releases. Merger `dev` → `master` au moment du bump de version.
- À chaque version : bumper la constante `Version` dans `internal/hfr/version.go` **et** ajouter une entrée dans `CHANGELOG.md`. Le User-Agent (`hfr-mcp/<Version>`) suit cette constante.
- Pousser un tag `vX.Y.Z` déclenche `release.yml` : cross-compilation (linux/darwin/windows × amd64/arm64) et publication de la release GitHub.

## Configuration et authentification (runtime)

- Identifiants HFR : variables d'env `HFR_LOGIN` / `HFR_PASSWD` (prioritaires), sinon `./hfr.conf` puis `~/.config/hfr/config` (format `login=` / `passwd=`).
- MCP : login *lazy* (connexion au premier appel d'outil), session conservée en mémoire pour la durée du process.
- Les outils d'écriture (`hfr_reply`, `hfr_edit`, `hfr_mp`, `hfr_create_topic`, `hfr_quote`) exigent l'authentification ; `hfr_read`, `hfr_topics` et `hfr_cats` fonctionnent en anonyme.
- **Garde-fou d'identité** (#32) : les écritures sont refusées par défaut tant qu'aucun compte attendu n'est déclaré (`HFR_EXPECT_LOGIN` / `expect_login=` côté serveur, `expect` MCP ou `--pseudo` CLI par appel). Comparaison par pseudo (casse ignorée) ou userId ; syntaxe typée `pseudo:` / `id:`. Opt-out historique : `HFR_ALLOW_UNGUARDED_WRITES=1`. `hfr_whoami` / `hfr whoami` expose le compte actif. L'autorité = le cookie `md_user` de session relu juste avant chaque POST.

## Conventions de contenu HFR

- Le contenu des messages est en **BBCode HFR** : `[b]`, `[url=]`, `[quotemsg=post,topic,userId]`, etc.
- Smileys : *builtin* (~25, syntaxe `:code:`) vs *perso* (syntaxe `[:pseudo]`) — deux syntaxes distinctes.
- **Pas d'emoji Unicode dans les posts** : un emoji placé après une balise peut tronquer le message côté HFR.
- `[fixed]` est un bloc, jamais inline — utiliser `[b]` pour de l'emphase inline.

## Pièges connus (vérifiés en conditions réelles)

- HFR renvoie un **HTML différent en authentifié vs anonyme** (le contenu lui-même, pas seulement les boutons) — tester les deux modes.
- `hfr_read` (`forum2.php`) peut servir un snapshot **périmé/caché** ; `hfr_quote` et `hfr_topics` renvoient l'état frais. Pour connaître le `userId` réel derrière un post, `hfr_quote` expose `[quotemsg=post,topic,userId]` (source de vérité, contrairement au libellé d'auteur de `hfr_read`).
- POST de réponse/édition : HFR exige le token anti-CSRF `hash_check` (récupéré au login). Un POST qui ne renvoie pas **tous** les champs attendus du formulaire peut modifier silencieusement des réglages du sujet (ex. l'abonnement aux notifications e-mail).

## Documentation utilisateur

Voir `README.md` (installation, configuration MCP, usage de la CLI, roadmap).
