# Changelog

## [1.1.0] - 2026-04-11

### Features
- `hfr_topics` / `hfr topics`: list topics in a category (#13)
- `hfr_cats` / `hfr cats`: list all categories and subcategories (#23)
- `hfr_create_topic` / `hfr new`: create a new topic in any category
- `--file` flag: read content from a file or stdin for new/reply/edit commands
- `-o` flag: write read/print output to a file instead of stdout/context (#22)
- Output file mode in MCP: `output` parameter on `hfr_read` writes to file and returns summary

### Infrastructure
- Release workflow: cross-compiled binaries for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64

### Docs
- README rewritten in French with Mermaid roadmap chart

## [1.0.0] - 2025-04-06

First stable release. Full MCP server + CLI for forum.hardware.fr.

### Features
- MCP server with 5 tools: `hfr_read`, `hfr_reply`, `hfr_edit`, `hfr_quote`, `hfr_mp`
- CLI binary with matching commands: read, print, reply, edit, quote, mp, version
- Print mode: ~1000 posts/page, no signatures (~4x lighter per post)
- Batch concurrent read: page ranges with goroutines (`from:to`, `last-N:last`)
- Last page resolution: `page=0` or `last` fetches the last page automatically
- Multiquote: cite multiple messages in one request via HFR cookie mechanism
- Content cleaning: strip signatures, edit notices, cite counters
- Config file support: `./hfr.conf` or `~/.config/hfr/config`, env vars override
- File permissions check: warns if config readable by others
- `--auth` flag for explicit login on read-only commands
- Lazy login on MCP: connection to HFR only on first tool call
- Custom User-Agent (`hfr-mcp/1.0.0`)
- `hfr version` / `hfr-mcp --version`

### Infrastructure
- CI: golangci-lint v2 + build both binaries (GitHub Actions)
- Dependabot: weekly Go module updates

### Bug fixes
- Success detection: handle feminine "postee" alongside "poste"
- Quote fetching: use `message.php` with `numrep` (not `forum2.php`)
- Pagination parsing: filter on `sujet_` prefix to avoid false positives
- Lazy login: don't block MCP handshake with login attempt

## [0.1.0] - 2025-04-05

Initial prototype. MCP server only, basic reply/edit/mp.
