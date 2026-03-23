# hfr-mcp

Serveur [MCP](https://modelcontextprotocol.io/) (Model Context Protocol) en Go pour interagir avec [forum.hardware.fr](https://forum.hardware.fr) depuis Claude Code.

## Outils

| Outil | Description |
|-------|-------------|
| `hfr_read` | Lire un topic (posts d'une page) |
| `hfr_reply` | Poster une reponse sur un topic |
| `hfr_edit` | Editer un post existant |
| `hfr_mp` | Envoyer un message prive |

Le contenu est en BBCode HFR (`[b]`, `[url=]`, `[quote]`, smileys `:o`, `[:pseudo]`, etc.).

## Installation

```bash
go build -o hfr-mcp ./cmd/hfr-mcp/
```

## Configuration Claude Code

Ajouter dans `.mcp.json` a la racine du projet :

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

Les variables `HFR_LOGIN` et `HFR_PASSWD` sont obligatoires. Le login est lazy : la connexion a HFR ne se fait qu'au premier appel d'outil.

## Architecture

```
cmd/hfr-mcp/main.go       Point d'entree, lazy login, stdio transport
internal/hfr/client.go     Client HTTP, login, hash_check, cookie jar
internal/hfr/post.go       Reply + Edit
internal/hfr/mp.go         Messages prives
internal/hfr/reader.go     Lecture de topics
internal/hfr/parser.go     Parsing HTML (goquery)
internal/hfr/errors.go     Types d'erreurs HFR
internal/mcp/tools.go      Declaration des 4 outils MCP
internal/mcp/format.go     Formatage des resultats
```

## Dependances

- [go-sdk/mcp](https://github.com/modelcontextprotocol/go-sdk) — SDK MCP officiel
- [goquery](https://github.com/PuerkitoBio/goquery) — Parsing HTML

## Licence

MIT
