# hfr-mcp

Outils Go pour interagir avec [forum.hardware.fr](https://forum.hardware.fr) : un serveur [MCP](https://modelcontextprotocol.io/) pour Claude Code et une CLI standalone.

## Fonctionnalites

| Action | MCP tool | CLI |
|--------|----------|-----|
| Lire un topic | `hfr_read` | `hfr read <cat> <post> [page]` |
| Poster une reponse | `hfr_reply` | `hfr reply <cat> <post> <content>` |
| Editer un post | `hfr_edit` | `hfr edit <cat> <post> <numreponse> <content>` |
| Citer un message | `hfr_quote` | `hfr quote <cat> <post> <numreponse>` |
| Envoyer un MP | `hfr_mp` | `hfr mp <dest> <subject> <content>` |

Le contenu est en BBCode HFR (`[b]`, `[url=]`, `[quotemsg=...]`, smileys `:o`, `[:pseudo]`, etc.).

## Installation

```bash
go build -o hfr-mcp ./cmd/hfr-mcp/   # serveur MCP
go build -o hfr ./cmd/hfr/            # CLI
```

## CLI

```bash
# Lecture anonyme
hfr read 13 120036 350

# Lecture authentifiee (acces topics restreints, etc.)
hfr --auth read 13 120036 350

# Poster (auth automatique)
hfr reply 13 120036 "Hello HFR :o"

# Citer un message (retourne le BBCode [quotemsg=...])
hfr quote 13 120036 74497677

# Editer
hfr edit 13 120036 74497677 "contenu modifie"

# Envoyer un MP
hfr mp pseudo "Sujet" "Corps du message"
```

Auth via variables d'environnement `HFR_LOGIN` et `HFR_PASSWD`. Les commandes qui ecrivent (reply, edit, quote, mp) exigent l'auth. `read` fonctionne en anonyme par defaut, `--auth` pour se connecter.

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

## Architecture

```
cmd/hfr/main.go            CLI : sous-commandes, parsing args, --auth
cmd/hfr-mcp/main.go        Serveur MCP : lazy login, stdio transport
internal/hfr/client.go     Client HTTP, login, hash_check, cookie jar
internal/hfr/post.go       Reply + Edit
internal/hfr/mp.go         Messages prives
internal/hfr/reader.go     Lecture de topics + FetchQuote (message.php)
internal/hfr/parser.go     Parsing HTML (goquery)
internal/hfr/models.go     Structs Post, Topic, EditInfo
internal/hfr/errors.go     Types d'erreurs HFR
internal/mcp/tools.go      Declaration des 5 outils MCP
internal/mcp/format.go     Formatage des resultats
```

## Dependances

- [go-sdk/mcp](https://github.com/modelcontextprotocol/go-sdk) — SDK MCP officiel
- [goquery](https://github.com/PuerkitoBio/goquery) — Parsing HTML

## Licence

MIT
