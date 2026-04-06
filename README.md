# hfr-mcp

Outils Go pour interagir avec [forum.hardware.fr](https://forum.hardware.fr) : un serveur [MCP](https://modelcontextprotocol.io/) pour Claude Code et une CLI standalone.

## Fonctionnalites

| Action | MCP tool | CLI |
|--------|----------|-----|
| Lire un topic | `hfr_read` | `hfr read <cat> <post> [page]` |
| Lire en mode print | `hfr_read` (print=true) | `hfr print <cat> <post> [page]` |
| Poster une reponse | `hfr_reply` | `hfr reply <cat> <post> <content>` |
| Editer un post | `hfr_edit` | `hfr edit <cat> <post> <numreponse> <content>` |
| Citer un message | `hfr_quote` | `hfr quote <cat> <post> <numreponse>` |
| Multiquote | `hfr_quote` (numreponses) | `hfr quote <cat> <post> <n1> <n2> ...` |
| Envoyer un MP | `hfr_mp` | `hfr mp <dest> <subject> <content>` |
| Version | — | `hfr version` / `hfr-mcp --version` |

Le contenu est en BBCode HFR (`[b]`, `[url=]`, `[quotemsg=...]`, smileys `:o`, `[:pseudo]`, etc.).

## Installation

```bash
go install github.com/XaaT/hfr-mcp/cmd/hfr-mcp@latest   # serveur MCP
go install github.com/XaaT/hfr-mcp/cmd/hfr@latest        # CLI
```

Ou depuis les sources :

```bash
go build -o hfr-mcp ./cmd/hfr-mcp/
go build -o hfr ./cmd/hfr/
```

## Configuration

Fichier de configuration (premier trouve) :

1. `./hfr.conf` (repertoire courant)
2. `~/.config/hfr/config`

Format :

```
login=pseudo
passwd=motdepasse
```

Les variables d'environnement `HFR_LOGIN` / `HFR_PASSWD` prennent le dessus sur le fichier de config.

Le fichier est verifie au demarrage : un warning s'affiche si les permissions sont trop ouvertes (lisible par d'autres utilisateurs).

## CLI

```bash
# Lecture anonyme
hfr read 13 120036 350

# Derniere page
hfr read 13 120036 last

# Range de pages (concurrent)
hfr read 13 120036 340:350

# Derniere page relative (les 5 dernieres pages)
hfr read 13 120036 last-4:last

# Mode print (~1000 posts/page, sans signatures, ~4x plus leger)
hfr print 13 120036
hfr print 13 120036 --last 20

# Lecture authentifiee
hfr --auth read 13 120036 350

# Poster (auth automatique)
hfr reply 13 120036 "Hello HFR :o"

# Citer un message (retourne le BBCode [quotemsg=...])
hfr quote 13 120036 74497677

# Multiquote
hfr quote 13 120036 74497677 74497680 74497685

# Editer
hfr edit 13 120036 74497677 "contenu modifie"

# Envoyer un MP
hfr mp pseudo "Sujet" "Corps du message"

# Version
hfr version
```

Les commandes d'ecriture (reply, edit, quote, mp) exigent l'authentification (automatique). `read` et `print` fonctionnent en anonyme par defaut, `--auth` pour se connecter.

## Configuration MCP (Claude Code)

Ajouter dans `.mcp.json` a la racine du projet, ou globalement via `claude mcp add --scope user` :

```json
{
  "mcpServers": {
    "hfr": {
      "command": "/chemin/vers/hfr-mcp",
      "env": {
        "HFR_LOGIN": "pseudo",
        "HFR_PASSWD": "motdepasse"
      }
    }
  }
}
```

Le login est lazy : la connexion a HFR ne se fait qu'au premier appel d'outil. La session est maintenue en memoire (cookie jar) pour toute la duree du process.

### Outils MCP

| Outil | Description |
|-------|-------------|
| `hfr_read` | Lire un topic. `page=0` derniere page, `page_from`/`page_to` pour du batch concurrent, `print=true` mode impression, `last=N` derniers posts |
| `hfr_reply` | Poster une reponse (BBCode) |
| `hfr_edit` | Editer un post existant |
| `hfr_quote` | Citer un ou plusieurs messages (`numreponse` ou `numreponses[]`) |
| `hfr_mp` | Envoyer un message prive |

## Optimisations token

- **Mode print** : `print=true` charge ~1000 posts/page au lieu de 40, sans signatures (~4x plus leger par post)
- **Content cleaning** : les signatures, notices d'edition, et compteurs de citation sont automatiquement supprimes
- **Batch concurrent** : les ranges de pages sont chargees en parallele (goroutines)
- **Pagination** : `TotalPages` retourne dans chaque reponse pour naviguer sans requetes supplementaires

## Architecture

```
cmd/hfr/main.go            CLI : sous-commandes, parsing args, --auth
cmd/hfr-mcp/main.go        Serveur MCP : lazy login, stdio transport
internal/hfr/client.go     Client HTTP, login, hash_check, cookie jar, User-Agent custom
internal/hfr/reader.go     Lecture de topics, print mode, batch concurrent, FetchQuote
internal/hfr/parser.go     Parsing HTML (goquery), content cleaning
internal/hfr/post.go       Reply + Edit
internal/hfr/mp.go         Messages prives
internal/hfr/models.go     Structs Post, Topic, EditInfo
internal/hfr/errors.go     Types d'erreurs HFR
internal/hfr/version.go    Constante de version
internal/config/config.go  Fichier de config + env vars + permissions check
internal/mcp/tools.go      Declaration des 5 outils MCP
internal/mcp/helpers.go    Formatage des resultats
```

Deux binaires separes : le CLI (`hfr`) et le serveur MCP (`hfr-mcp`) ont des cycles de vie differents et ne partagent pas les memes dependances de build.

## Dependances

- [go-sdk/mcp](https://github.com/modelcontextprotocol/go-sdk) — SDK MCP officiel
- [goquery](https://github.com/PuerkitoBio/goquery) — Parsing HTML

## Licence

MIT
