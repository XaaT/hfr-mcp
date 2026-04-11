# hfr-mcp-action Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a GitHub Action `XaaT/hfr-mcp-action` that lets anyone sync BBCode files to HFR posts on push — "FP as Code" MVP.

**Architecture:** Composite GitHub Action that downloads a pre-built `hfr` CLI binary from GitHub releases, reads a mapping config (YAML) or direct inputs, and calls `hfr --auth edit --file` for each file→post pair. Prerequisites: `--file` flag on the CLI + release workflow with cross-compiled binaries.

**Tech Stack:** Bash (composite action), Go (CLI change + release workflow), YAML config, GitHub Actions

---

## File Structure

### hfr-mcp repo (prerequisite changes)

| File | Action | Purpose |
|------|--------|---------|
| `cmd/hfr/main.go` | Modify | Add `--file` flag to `edit` and `reply` commands |
| `.github/workflows/release.yml` | Create | Build + attach cross-compiled binaries on tag push |

### New repo: XaaT/hfr-mcp-action

| File | Action | Purpose |
|------|--------|---------|
| `action.yml` | Create | Composite action definition with inputs |
| `sync.sh` | Create | Entrypoint: parse config or inputs, call hfr edit per post |
| `README.md` | Create | French docs with examples |
| `LICENSE` | Create | MIT |

---

### Task 1: Add `--file` flag to hfr CLI edit command

**Why:** Passing large BBCode content as a shell argument is fragile — special characters (`$`, `` ` ``, `"`, `!`) can break shell expansion. Reading from a file is robust and has no size limit.

**Files:**
- Modify: `cmd/hfr/main.go:222-235` (cmdEdit function)

- [ ] **Step 1: Add `--file` parsing to cmdEdit**

In `cmd/hfr/main.go`, replace `cmdEdit`:

```go
func cmdEdit(client *hfr.Client, args []string) {
	if len(args) < 3 {
		die("usage: hfr edit <cat> <post> <numreponse> [content | --file <path>]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	numreponse := mustInt(args[2], "numreponse")

	content := readContent(args[3:])

	if err := client.Edit(cat, post, numreponse, content); err != nil {
		die("edit failed: %v", err)
	}
	fmt.Println("Post edited.")
}
```

- [ ] **Step 2: Add `--file` parsing to cmdReply**

Replace `cmdReply`:

```go
func cmdReply(client *hfr.Client, args []string) {
	if len(args) < 2 {
		die("usage: hfr reply <cat> <post> [content | --file <path>]")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")

	content := readContent(args[2:])

	if err := client.Reply(cat, post, content); err != nil {
		die("reply failed: %v", err)
	}
	fmt.Println("Reply posted.")
}
```

- [ ] **Step 3: Add readContent helper**

Add at the bottom of `cmd/hfr/main.go`, before the existing helpers:

```go
func readContent(args []string) string {
	if len(args) >= 2 && args[0] == "--file" {
		path := args[1]
		var data []byte
		var err error
		if path == "-" {
			data, err = io.ReadAll(os.Stdin)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			die("read file failed: %v", err)
		}
		return strings.TrimRight(string(data), "\n")
	}
	if len(args) == 0 {
		die("content required: provide text or --file <path>")
	}
	return strings.Join(args, " ")
}
```

Add `"io"` to imports.

- [ ] **Step 4: Update usage string**

In the `usage` const, update edit and reply lines:

```
  reply    <cat> <post> <content|--file path>   Post a reply
  edit     <cat> <post> <numreponse> <content|--file path>  Edit a post
```

- [ ] **Step 5: Test locally**

```bash
# Test --file with a temp file
echo "[b]Test FP[/b]" > /tmp/test-fp.bbcode
go run ./cmd/hfr/ edit 13 12345 0 --file /tmp/test-fp.bbcode
# Expected: "credentials required" (no login, but proves parsing works)

# Test --file - (stdin)
echo "[b]Test FP[/b]" | go run ./cmd/hfr/ edit 13 12345 0 --file -
# Expected: same "credentials required"

# Test backward compat (inline content)
go run ./cmd/hfr/ edit 13 12345 0 "hello world"
# Expected: same "credentials required"
```

- [ ] **Step 6: Commit**

```bash
git add cmd/hfr/main.go
git commit -m "feat(cli): add --file flag to edit and reply commands

Allows reading post content from a file or stdin instead of
command-line arguments. Needed for GitHub Action integration
where BBCode content may contain shell-unsafe characters."
```

---

### Task 2: Add release workflow to hfr-mcp

**Why:** The GitHub Action needs a pre-built binary to avoid installing Go on the runner (~40s overhead). Cross-compile 5 targets on tag push.

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write release workflow**

```yaml
name: Release

on:
  push:
    tags: ['v*']

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build binaries
        run: |
          mkdir -p dist
          targets="linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64"
          for target in $targets; do
            os="${target%/*}"
            arch="${target#*/}"
            ext=""
            [ "$os" = "windows" ] && ext=".exe"
            echo "Building $os/$arch..."
            GOOS=$os GOARCH=$arch go build -ldflags="-s -w" \
              -o "dist/hfr-${os}-${arch}${ext}" ./cmd/hfr/
            GOOS=$os GOARCH=$arch go build -ldflags="-s -w" \
              -o "dist/hfr-mcp-${os}-${arch}${ext}" ./cmd/hfr-mcp/
          done

      - name: Create release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          files: dist/*
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: add release workflow with cross-compiled binaries

Builds hfr and hfr-mcp for linux/amd64, linux/arm64,
darwin/amd64, darwin/arm64, windows/amd64 on tag push.
Uses softprops/action-gh-release for asset upload."
```

---

### Task 3: Create repo XaaT/hfr-mcp-action

- [ ] **Step 1: Create the repo on GitHub**

```bash
gh repo create XaaT/hfr-mcp-action \
  --public \
  --description "GitHub Action pour synchroniser du contenu BBCode vers HFR — FP as Code" \
  --clone
```

- [ ] **Step 2: Configure git identity**

```bash
cd /work/xaat/hfr-mcp-action
git config user.name "xat"
git config user.email "xat@azora.fr"
```

---

### Task 4: Write action.yml

**Files:**
- Create: `action.yml`

- [ ] **Step 1: Write action.yml**

```yaml
name: 'HFR Sync'
description: 'Synchronise des fichiers BBCode vers des posts HFR — FP as Code'
author: 'XaaT'

branding:
  icon: 'upload-cloud'
  color: 'orange'

inputs:
  file:
    description: 'Fichier BBCode à synchroniser'
    required: false
  cat:
    description: 'Catégorie HFR'
    required: false
  post:
    description: 'ID du topic HFR'
    required: false
  numreponse:
    description: 'Numéro du post à éditer (0 = premier post)'
    required: false
    default: '0'
  config:
    description: 'Chemin vers hfr-sync.yaml (mode multi-posts)'
    required: false
  dry-run:
    description: 'Afficher les actions sans les exécuter'
    required: false
    default: 'false'
  hfr-login:
    description: 'Login HFR'
    required: true
  hfr-passwd:
    description: 'Mot de passe HFR'
    required: true
  hfr-version:
    description: 'Version du CLI hfr (tag sans le v, ex: 1.0.0)'
    required: false
    default: 'latest'

runs:
  using: 'composite'
  steps:
    - name: Detect version
      id: version
      shell: bash
      run: |
        version="${{ inputs.hfr-version }}"
        if [ "$version" = "latest" ]; then
          version=$(gh release view --repo XaaT/hfr-mcp --json tagName -q .tagName | sed 's/^v//')
        fi
        echo "version=$version" >> "$GITHUB_OUTPUT"
      env:
        GH_TOKEN: ${{ github.token }}

    - name: Download hfr CLI
      shell: bash
      run: |
        url="https://github.com/XaaT/hfr-mcp/releases/download/v${{ steps.version.outputs.version }}/hfr-linux-amd64"
        echo "Downloading $url"
        curl -fsSL "$url" -o /usr/local/bin/hfr
        chmod +x /usr/local/bin/hfr
        hfr version

    - name: Sync to HFR
      shell: bash
      run: bash ${{ github.action_path }}/sync.sh
      env:
        HFR_LOGIN: ${{ inputs.hfr-login }}
        HFR_PASSWD: ${{ inputs.hfr-passwd }}
        INPUT_FILE: ${{ inputs.file }}
        INPUT_CAT: ${{ inputs.cat }}
        INPUT_POST: ${{ inputs.post }}
        INPUT_NUMREPONSE: ${{ inputs.numreponse }}
        INPUT_CONFIG: ${{ inputs.config }}
        INPUT_DRY_RUN: ${{ inputs.dry-run }}
```

- [ ] **Step 2: Commit**

```bash
git add action.yml
git commit -m "feat: add composite action definition

Downloads pre-built hfr binary from releases instead of
compiling from source. Supports simple mode (one file) and
config mode (hfr-sync.yaml with multiple posts)."
```

---

### Task 5: Write sync.sh

**Files:**
- Create: `sync.sh`

- [ ] **Step 1: Write sync.sh**

```bash
#!/usr/bin/env bash
set -euo pipefail

# ── helpers ──────────────────────────────────────────────
log()    { echo "::group::$1"; }
endlog() { echo "::endgroup::"; }
info()   { echo "$*"; }
err()    { echo "::error::$*"; exit 1; }

sync_post() {
  local file="$1" cat="$2" post="$3" numreponse="$4"

  [ -f "$file" ] || err "fichier introuvable: $file"

  info "sync $file -> cat=$cat post=$post numreponse=$numreponse"

  if [ "$INPUT_DRY_RUN" = "true" ]; then
    echo "DRY-RUN: hfr --auth edit $cat $post $numreponse --file $file"
    echo "Contenu (5 premieres lignes):"
    head -5 "$file"
    return
  fi

  hfr --auth edit "$cat" "$post" "$numreponse" --file "$file"
  info "OK"
}

# ── mode simple ──────────────────────────────────────────
if [ -n "${INPUT_FILE:-}" ]; then
  [ -n "${INPUT_CAT:-}" ]  || err "input 'cat' requis en mode simple"
  [ -n "${INPUT_POST:-}" ] || err "input 'post' requis en mode simple"

  log "Sync: $INPUT_FILE"
  sync_post "$INPUT_FILE" "$INPUT_CAT" "$INPUT_POST" "${INPUT_NUMREPONSE:-0}"
  endlog
  exit 0
fi

# ── mode config ──────────────────────────────────────────
if [ -n "${INPUT_CONFIG:-}" ]; then
  [ -f "$INPUT_CONFIG" ] || err "config introuvable: $INPUT_CONFIG"

  command -v yq >/dev/null 2>&1 || {
    echo "Installing yq..."
    curl -fsSL "https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64" \
      -o /usr/local/bin/yq
    chmod +x /usr/local/bin/yq
  }

  cat_default=$(yq -r '.topic.cat' "$INPUT_CONFIG")
  post_default=$(yq -r '.topic.post' "$INPUT_CONFIG")
  count=$(yq -r '.topic.posts | length' "$INPUT_CONFIG")

  log "Config: $INPUT_CONFIG ($count posts)"

  for i in $(seq 0 $((count - 1))); do
    file=$(yq -r ".topic.posts[$i].file" "$INPUT_CONFIG")
    cat=$(yq -r ".topic.posts[$i].cat // $cat_default" "$INPUT_CONFIG")
    post=$(yq -r ".topic.posts[$i].post // $post_default" "$INPUT_CONFIG")
    numreponse=$(yq -r ".topic.posts[$i].numreponse // 0" "$INPUT_CONFIG")
    sync_post "$file" "$cat" "$post" "$numreponse"
  done

  endlog
  exit 0
fi

err "un des inputs 'file' ou 'config' est requis"
```

- [ ] **Step 2: Make executable and commit**

```bash
chmod +x sync.sh
git add sync.sh
git commit -m "feat: add sync entrypoint script

Two modes: simple (one file via inputs) and config
(hfr-sync.yaml for multi-post sync). Auto-installs yq
for YAML parsing in config mode."
```

---

### Task 6: Write README.md

**Files:**
- Create: `README.md`

- [ ] **Step 1: Write README.md**

````markdown
# hfr-mcp-action

GitHub Action pour synchroniser du contenu BBCode vers des posts HFR.
Maintenez vos First Posts dans Git, mettez-les a jour automatiquement a chaque push.

## Utilisation rapide

```yaml
- uses: XaaT/hfr-mcp-action@v0.1
  with:
    file: fp/main.bbcode
    cat: '13'
    post: '12345'
    numreponse: '0'
    hfr-login: ${{ secrets.HFR_LOGIN }}
    hfr-passwd: ${{ secrets.HFR_PASSWD }}
```

## Mode multi-posts

Creez un fichier `hfr-sync.yaml` :

```yaml
topic:
  cat: 13
  post: 12345
  posts:
    - file: fp/main.bbcode
      numreponse: 0
    - file: fp/reserve1.bbcode
      numreponse: 56789
```

```yaml
- uses: XaaT/hfr-mcp-action@v0.1
  with:
    config: hfr-sync.yaml
    hfr-login: ${{ secrets.HFR_LOGIN }}
    hfr-passwd: ${{ secrets.HFR_PASSWD }}
```

## Workflow complet

```yaml
name: Sync FP to HFR
on:
  push:
    branches: [main]
    paths: ['fp/**']

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: XaaT/hfr-mcp-action@v0.1
        with:
          config: hfr-sync.yaml
          hfr-login: ${{ secrets.HFR_LOGIN }}
          hfr-passwd: ${{ secrets.HFR_PASSWD }}
```

## Inputs

| Input | Requis | Defaut | Description |
|-------|--------|--------|-------------|
| `file` | non* | | Fichier BBCode a synchroniser |
| `cat` | non* | | Categorie HFR |
| `post` | non* | | ID du topic |
| `numreponse` | non | `0` | Numero du post (0 = premier post) |
| `config` | non* | | Chemin vers `hfr-sync.yaml` |
| `dry-run` | non | `false` | Previsualiser sans poster |
| `hfr-login` | oui | | Login HFR |
| `hfr-passwd` | oui | | Mot de passe HFR |
| `hfr-version` | non | `latest` | Version du CLI hfr |

\* Un des deux modes est requis : `file` + `cat` + `post`, ou `config`.

## Securite

- Utilisez **toujours** des [GitHub Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets) pour `hfr-login` et `hfr-passwd`
- Activez `dry-run: 'true'` pour tester avant de poster pour de vrai
- Limitez le workflow aux branches protegees pour eviter les posts accidentels

## Licence

MIT
````

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add French README with usage examples"
```

---

### Task 7: Add LICENSE, tag and release

**Files:**
- Create: `LICENSE`

- [ ] **Step 1: Create MIT license**

```bash
cat > LICENSE << 'EOF'
MIT License

Copyright (c) 2026 XaaT

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
EOF
```

- [ ] **Step 2: Commit**

```bash
git add LICENSE
git commit -m "chore: add MIT license"
```

- [ ] **Step 3: Push and tag**

```bash
git push -u origin main
git tag v0.1.0
git push origin v0.1.0
```

- [ ] **Step 4: Create release**

```bash
gh release create v0.1.0 \
  -R XaaT/hfr-mcp-action \
  --title "v0.1.0 — FP as Code MVP" \
  --notes "Premier release. Synchronise des fichiers BBCode vers des posts HFR via GitHub Actions."
```

---

## Dependency Graph

```
Task 1 (--file flag in hfr-mcp) ─┐
Task 2 (release workflow)     ────┤ (hfr-mcp, sequential)
                                  │
                                  └─→ hfr-mcp release with binaries
                                       │
                                       └─→ Task 3 (create repo)
                                             ├─→ Task 4 (action.yml) ──┐
                                             ├─→ Task 5 (sync.sh)  ────┤ (parallel)
                                             ├─→ Task 6 (README.md) ───┘
                                             └─→ Task 7 (LICENSE + tag + release)
```

Tasks 1-2 must be merged and released in hfr-mcp (with a new tag) before the action can work. Tasks 4, 5, 6 are independent and can be done in parallel.
