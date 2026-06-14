# HFR Identity Guard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refuse HFR writes when the logged-in account doesn't match the expected account, and expose the active account, so messages never go out under the wrong identity.

**Architecture:** All logic lives in the `hfr.Client`. Writes go through one atomic path (`authenticatedPost`) that resolves the *current* identity from the `md_user` cookie (the real POST authority), checks it against the expected account (server `HFR_EXPECT_LOGIN` and per-call `expect`), and only then POSTs. Fail-closed by default unless `HFR_ALLOW_UNGUARDED_WRITES=1`. CLI and MCP only feed constraints and format the returned `Identity`.

**Tech Stack:** Go 1.25, `goquery`, `modelcontextprotocol/go-sdk`, `net/http/httptest` for tests.

**Spec:** `/work/xaat/hfr-mcp/docs/superpowers/specs/2026-06-14-hfr-identity-guard-design.md`

**Working branch:** create `feat/32-identity-guard` off `dev` before Task 1 (`git switch -c feat/32-identity-guard`). Commit identity is the repo-local `xat <xat@azora.fr>` (already configured). Each commit message ends with the `Co-Authored-By` trailer.

**Validate-before-push (per AGENTS.md):** `go vet ./...`, `golangci-lint run`, `go build ./...`, `go test ./...`.

---

## Task 1: Identity value type + matching logic (pure)

**Files:**
- Create: `internal/hfr/identity.go`
- Modify: `internal/hfr/errors.go`
- Test: `internal/hfr/identity_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/hfr/identity_test.go`:

```go
package hfr

import "testing"

func TestIdentityMatches(t *testing.T) {
	tests := []struct {
		name    string
		id      Identity
		want    string
		ok      bool
		wantErr bool
	}{
		{"pseudo exact", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "xatelitte", true, false},
		{"pseudo case+space", Identity{Pseudo: "xatelitte"}, "  XaTeLitte ", true, false},
		{"pseudo mismatch", Identity{Pseudo: "XaTriX"}, "xatelitte", false, false},
		{"pseudo prefix", Identity{Pseudo: "xatelitte"}, "pseudo:xatelitte", true, false},
		{"id prefix match", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "id:1214571", true, false},
		{"id prefix mismatch", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "id:54596", false, false},
		{"bare numeric -> userId", Identity{Pseudo: "xatelitte", UserID: "1214571"}, "1214571", true, false},
		{"numeric pseudo via prefix", Identity{Pseudo: "1234"}, "pseudo:1234", true, false},
		{"id wanted but unresolved", Identity{Pseudo: "xatelitte", UserID: ""}, "id:1214571", false, true},
		{"zero is a real constraint", Identity{Pseudo: "x", UserID: "5"}, "0", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ok, err := identityMatches(tc.id, tc.want)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hfr/ -run TestIdentityMatches -v`
Expected: FAIL — `undefined: Identity` / `undefined: identityMatches`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/hfr/identity.go`:

```go
package hfr

import "strings"

// Identity is the account currently behind the session.
type Identity struct {
	Pseudo        string
	UserID        string
	Authenticated bool
}

// normalize folds case and trims surrounding space for pseudo comparison.
func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// parseExpect splits a typed constraint.
// "id:123" -> (true,"123"); "pseudo:xat" -> (false,"xat");
// bare all-digits -> (true, digits); otherwise (false, trimmed).
func parseExpect(want string) (isID bool, value string) {
	w := strings.TrimSpace(want)
	switch {
	case strings.HasPrefix(w, "id:"):
		return true, strings.TrimSpace(strings.TrimPrefix(w, "id:"))
	case strings.HasPrefix(w, "pseudo:"):
		return false, strings.TrimSpace(strings.TrimPrefix(w, "pseudo:"))
	default:
		if isAllDigits(w) {
			return true, w
		}
		return false, w
	}
}

// identityMatches reports whether id satisfies the constraint want.
// It errors if want targets a userId the session could not resolve.
func identityMatches(id Identity, want string) (bool, error) {
	isID, val := parseExpect(want)
	if isID {
		if id.UserID == "" {
			return false, &HfrError{Code: "identity", Message: "expected account is a userId but the session userId could not be resolved"}
		}
		return val == id.UserID, nil
	}
	return normalize(val) == normalize(id.Pseudo), nil
}
```

Add to `internal/hfr/errors.go` inside the `var (...)` block:

```go
	ErrNoExpectedAccount = &HfrError{Code: "identity", Message: "write refused: no expected account configured (set HFR_EXPECT_LOGIN / --pseudo, or HFR_ALLOW_UNGUARDED_WRITES=1)"}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hfr/ -run TestIdentityMatches -v`
Expected: PASS (all subtests).

- [ ] **Step 5: Commit**

```bash
git add internal/hfr/identity.go internal/hfr/identity_test.go internal/hfr/errors.go
git commit -m "feat(hfr): identity type + typed expect matching (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: Guard fields on Client + checkIdentity (fail-closed)

**Files:**
- Modify: `internal/hfr/client.go` (struct + setters)
- Modify: `internal/hfr/identity.go` (add `checkIdentity`)
- Test: `internal/hfr/identity_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/hfr/identity_test.go`:

```go
func TestCheckIdentity(t *testing.T) {
	id := Identity{Pseudo: "xatelitte", UserID: "1214571", Authenticated: true}

	t.Run("no constraint, fail-closed", func(t *testing.T) {
		c := &Client{}
		if err := c.checkIdentity(id, ""); err != ErrNoExpectedAccount {
			t.Fatalf("got %v, want ErrNoExpectedAccount", err)
		}
	})
	t.Run("no constraint, opt-out allows", func(t *testing.T) {
		c := &Client{allowUnguarded: true}
		if err := c.checkIdentity(id, ""); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})
	t.Run("server constraint match", func(t *testing.T) {
		c := &Client{expectedLogin: "xatelitte"}
		if err := c.checkIdentity(id, ""); err != nil {
			t.Fatalf("got %v, want nil", err)
		}
	})
	t.Run("server constraint mismatch", func(t *testing.T) {
		c := &Client{expectedLogin: "XaTriX"}
		err := c.checkIdentity(id, "")
		if he, ok := err.(*HfrError); !ok || he.Code != "identity" {
			t.Fatalf("got %v, want identity error", err)
		}
	})
	t.Run("call constraint mismatch beats matching server", func(t *testing.T) {
		c := &Client{expectedLogin: "xatelitte"}
		if err := c.checkIdentity(id, "id:54596"); err == nil {
			t.Fatal("expected mismatch error")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hfr/ -run TestCheckIdentity -v`
Expected: FAIL — `c.checkIdentity undefined`, `unknown field allowUnguarded/expectedLogin`.

- [ ] **Step 3: Write minimal implementation**

In `internal/hfr/client.go`, extend the `Client` struct (add the four fields + mutex; keep existing fields):

```go
import (
	// ...existing imports...
	"sync"
)

type Client struct {
	http           *http.Client
	ua             string
	pseudo         string
	hashCheck      string
	authed         bool
	userID         string // resolved numeric HFR user id ("" if unknown)
	expectedLogin  string // server-side expected account (HFR_EXPECT_LOGIN)
	allowUnguarded bool   // HFR_ALLOW_UNGUARDED_WRITES opt-out
	baseURL        string // injectable; defaults to defaultBaseURL
	mu             sync.Mutex
}
```

Add setters near `NewClient` in `client.go`:

```go
// SetExpectedLogin sets the server-side expected account guard.
func (c *Client) SetExpectedLogin(login string) { c.expectedLogin = login }

// SetAllowUnguarded enables writes when no expected account is configured.
func (c *Client) SetAllowUnguarded(b bool) { c.allowUnguarded = b }
```

Add `checkIdentity` to `internal/hfr/identity.go`:

```go
import "fmt" // add to identity.go imports

// checkIdentity enforces the expected-account constraints against id.
// Fail-closed: with no constraint and no opt-out it refuses.
func (c *Client) checkIdentity(id Identity, expect string) error {
	var constraints []string
	if c.expectedLogin != "" {
		constraints = append(constraints, c.expectedLogin)
	}
	if expect != "" {
		constraints = append(constraints, expect)
	}
	if len(constraints) == 0 {
		if c.allowUnguarded {
			return nil
		}
		return ErrNoExpectedAccount
	}
	for _, want := range constraints {
		ok, err := identityMatches(id, want)
		if err != nil {
			return err
		}
		if !ok {
			return &HfrError{Code: "identity", Message: fmt.Sprintf(
				"write refused: connected as %q (userId %s) != expected %q", id.Pseudo, id.UserID, want)}
		}
	}
	return nil
}
```

> Note: `identity.go` now imports both `strings` and `fmt`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/hfr/ -run 'TestCheckIdentity|TestIdentityMatches' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hfr/client.go internal/hfr/identity.go internal/hfr/identity_test.go
git commit -m "feat(hfr): fail-closed identity guard on Client (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: Make baseURL injectable (testability — review #9)

**Files:**
- Modify: `internal/hfr/client.go` (const rename, `NewClient`, `doPost`, `fetchHashCheck`)
- Modify: `internal/hfr/reader.go` (`readPage`, `ListTopics`, `FetchQuote`)
- Modify: `internal/hfr/post.go` (`Edit` editURL)

- [ ] **Step 1: Rename the const and seed the field**

In `client.go` change line 14 from:

```go
const baseURL = "https://forum.hardware.fr"
```
to:
```go
const defaultBaseURL = "https://forum.hardware.fr"
```

In `NewClient` set the field (add `baseURL: defaultBaseURL,` to the returned struct literal):

```go
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		ua:      "hfr-mcp/" + Version,
		baseURL: defaultBaseURL,
	}
}
```

- [ ] **Step 2: Replace every `baseURL` reference with `c.baseURL`**

These are all methods on `*Client`, so `c.baseURL` is in scope. Replace:

- `client.go` Login cookie check: `u, _ := url.Parse(baseURL)` → `u, _ := url.Parse(c.baseURL)`
- `client.go` `fetchHashCheck`: `c.doGet(baseURL + "/user/editprofil.php?config=hardwarefr.inc")` → `c.doGet(c.baseURL + "/user/editprofil.php?config=hardwarefr.inc")`
- `client.go` `doPost`: `http.NewRequest("POST", baseURL+endpoint, body)` → `http.NewRequest("POST", c.baseURL+endpoint, body)`
- `reader.go` `readPage` Sprintf: `baseURL` → `c.baseURL`
- `reader.go` `ListTopics` Sprintf: `baseURL` → `c.baseURL`
- `reader.go` `FetchQuote`: `url.Parse(baseURL)` → `url.Parse(c.baseURL)` AND quoteURL Sprintf `baseURL` → `c.baseURL`
- `post.go` `Edit` editURL Sprintf: `baseURL` → `c.baseURL`

- [ ] **Step 3: Verify the rename is complete**

Run: `grep -rn '\bbaseURL\b' internal/hfr/`
Expected: only `defaultBaseURL` (the const) and `c.baseURL` references — no bare `baseURL` left.

- [ ] **Step 4: Build + existing tests**

Run: `go build ./... && go test ./internal/hfr/ -v`
Expected: build OK, Task 1–2 tests still PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hfr/client.go internal/hfr/reader.go internal/hfr/post.go
git commit -m "refactor(hfr): make baseURL injectable for tests (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: Login rollback, userId resolution, currentIdentity, Whoami (review #1, #3, #4, #8)

**Files:**
- Modify: `internal/hfr/client.go` (`Login`, replace `fetchHashCheck` with `fetchProfile`, add `currentIdentity`, `Whoami`, `parseUserID`)
- Test: `internal/hfr/login_test.go`

> **Live verification required first.** The exact location of the numeric userId on the authenticated `editprofil` page is not yet confirmed. Before writing `parseUserID`, fetch the real page once with valid creds and inspect it:
>
> ```bash
> go build -o /tmp/hfr ./cmd/hfr/
> HFR_LOGIN=… HFR_PASSWD=… /tmp/hfr --auth whoami   # (whoami lands in Task 8; for now:)
> # quick probe: log the editprofil HTML + cookies during a login from a scratch main, OR
> # inspect cookies: look for a cookie named like "md_user_id"; if absent, grep the page for
> # the self profile link (profil.php?...user=NNNN / user_id=NNNN) or a hidden input.
> ```
>
> Lock the selector/cookie name from what you observe, then build the test fixture in Step 1 from the *real* markup. The strategy below is cookie-first with an HTML fallback; adjust the fallback selector to match reality.

- [ ] **Step 1: Write the failing test**

Create `internal/hfr/login_test.go`:

```go
package hfr

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestClient points a Client at a test server.
func newTestClient(baseURL string) *Client {
	c := NewClient()
	c.baseURL = baseURL
	return c
}

// loginMux serves a minimal HFR login + profile. userIDInPage controls whether
// the editprofil page exposes the numeric id (fallback path).
func loginMux(t *testing.T, pseudo, userID string, hashOK, userIDInPage bool) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: pseudo, Path: "/"})
		if userID != "" {
			http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: userID, Path: "/"})
		}
		_, _ = w.Write([]byte("<html><body>ok</body></html>"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		if !hashOK {
			_, _ = w.Write([]byte("<html><body>no token</body></html>"))
			return
		}
		page := `<html><body><input type="hidden" name="hash_check" value="HASH123">`
		if userIDInPage {
			page += `<a href="/user/profil.php?config=hfr.inc&amp;user=` + userID + `">profil</a>`
		}
		page += `</body></html>`
		_, _ = w.Write([]byte(page))
	})
	return httptest.NewServer(mux)
}

func TestLoginResolvesIdentity(t *testing.T) {
	srv := loginMux(t, "xatelitte", "1214571", true, true)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := c.currentIdentity()
	if err != nil {
		t.Fatalf("currentIdentity: %v", err)
	}
	if id.Pseudo != "xatelitte" || id.UserID != "1214571" || !id.Authenticated {
		t.Fatalf("identity = %+v", id)
	}
}

func TestLoginRollbackOnHashFailure(t *testing.T) {
	srv := loginMux(t, "xatelitte", "1214571", false, false)
	defer srv.Close()
	c := newTestClient(srv.URL)
	if err := c.Login("xatelitte", "pw"); err == nil {
		t.Fatal("expected login error when hash_check missing")
	}
	if c.authed {
		t.Fatal("authed must stay false after failed login")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hfr/ -run TestLogin -v`
Expected: FAIL — `c.currentIdentity undefined`; rollback test fails because current `Login` sets `authed=true` before the hash step.

- [ ] **Step 3: Write minimal implementation**

In `client.go`, rewrite `Login` to commit state only on full success, and replace `fetchHashCheck` with `fetchProfile`:

```go
func (c *Client) Login(pseudo, password string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data := url.Values{"pseudo": {pseudo}, "password": {password}}
	doc, err := c.doPost("/login_validation.php?config=hfr.inc", data)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	if strings.Contains(doc.Text(), "Votre mot de passe ou nom d'utilisateur n'est pas valide") {
		return ErrInvalidCreds
	}

	u, _ := url.Parse(c.baseURL)
	found := false
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "md_user" && cookie.Value == pseudo {
			found = true
		}
	}
	if !found {
		return &HfrError{Code: "auth", Message: "login failed: md_user cookie not set"}
	}

	hash, userID, err := c.fetchProfile()
	if err != nil {
		return err // no state mutated yet
	}

	c.pseudo = pseudo
	c.hashCheck = hash
	c.userID = userID
	c.authed = true
	return nil
}

// fetchProfile loads the authenticated profile page and extracts the
// anti-CSRF token and (best-effort) the numeric user id.
func (c *Client) fetchProfile() (hash, userID string, err error) {
	doc, err := c.doGet(c.baseURL + "/user/editprofil.php?config=hardwarefr.inc")
	if err != nil {
		return "", "", fmt.Errorf("profile fetch failed: %w", err)
	}
	hash, ok := doc.Find("input[name=hash_check]").Attr("value")
	if !ok || hash == "" {
		return "", "", ErrNoHashCheck
	}
	return hash, c.parseUserID(doc), nil
}

// parseUserID resolves the numeric user id, cookie first, page fallback.
// Returns "" if it cannot be resolved (guard then refuses id: constraints).
func (c *Client) parseUserID(doc *goquery.Document) string {
	u, _ := url.Parse(c.baseURL)
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "md_user_id" && ck.Value != "" {
			return ck.Value
		}
	}
	// Fallback: a self profile link carrying user=NNNN (adjust to live markup).
	id := ""
	doc.Find(`a[href*="user="]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if i := strings.Index(href, "user="); i >= 0 {
			rest := href[i+len("user="):]
			j := 0
			for j < len(rest) && rest[j] >= '0' && rest[j] <= '9' {
				j++
			}
			if j > 0 {
				id = rest[:j]
				return false
			}
		}
		return true
	})
	return id
}

// currentIdentity reads the account behind the live session (cookie authority).
func (c *Client) currentIdentity() (Identity, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return Identity{}, err
	}
	pseudo := ""
	for _, ck := range c.http.Jar.Cookies(u) {
		if ck.Name == "md_user" {
			if v, derr := url.QueryUnescape(ck.Value); derr == nil {
				pseudo = v
			} else {
				pseudo = ck.Value
			}
		}
	}
	if pseudo == "" {
		return Identity{}, ErrNotAuthenticated
	}
	return Identity{Pseudo: pseudo, UserID: c.userID, Authenticated: true}, nil
}

// Whoami returns the account currently behind the session.
func (c *Client) Whoami() (Identity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentIdentity()
}
```

Delete the old `fetchHashCheck` method (replaced by `fetchProfile`). Ensure `goquery` is imported in `client.go` (it already is).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/hfr/ -run TestLogin -v && go build ./...`
Expected: PASS, build OK.

- [ ] **Step 5: Commit**

```bash
git add internal/hfr/client.go internal/hfr/login_test.go
git commit -m "feat(hfr): resolve identity at login, rollback on failure (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: Atomic write path + rewire write methods (review #1, #2, #10)

**Files:**
- Modify: `internal/hfr/post.go` (`Reply`, `CreateTopic`, `Edit`)
- Modify: `internal/hfr/mp.go` (`SendMP`)
- Modify: `internal/hfr/client.go` (add `authenticatedPost`)
- Test: `internal/hfr/write_guard_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/hfr/write_guard_test.go`:

```go
package hfr

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// writeMux extends loginMux with a reply endpoint that counts POSTs.
func writeServer(t *testing.T, pseudo, userID string, posted *int32) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/login_validation.php", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "md_user", Value: pseudo, Path: "/"})
		http.SetCookie(w, &http.Cookie{Name: "md_user_id", Value: userID, Path: "/"})
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/user/editprofil.php", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<input type="hidden" name="hash_check" value="H">`))
	})
	mux.HandleFunc("/bddpost.php", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(posted, 1)
		_, _ = w.Write([]byte("Votre message a été posté avec succès"))
	})
	return httptest.NewServer(mux)
}

func TestReplyRefusedOnMismatchNoPost(t *testing.T) {
	var posted int32
	srv := writeServer(t, "XaTriX", "54596", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("XaTriX", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	_, err := c.Reply(23, 35421, "hi", "")
	if err == nil {
		t.Fatal("expected identity refusal")
	}
	if atomic.LoadInt32(&posted) != 0 {
		t.Fatalf("POST must not be sent on mismatch, got %d", posted)
	}
}

func TestReplyAllowedReturnsIdentity(t *testing.T) {
	var posted int32
	srv := writeServer(t, "xatelitte", "1214571", &posted)
	defer srv.Close()
	c := newTestClient(srv.URL)
	c.SetExpectedLogin("xatelitte")
	if err := c.Login("xatelitte", "pw"); err != nil {
		t.Fatalf("login: %v", err)
	}
	id, err := c.Reply(23, 35421, "hi", "id:1214571")
	if err != nil {
		t.Fatalf("reply: %v", err)
	}
	if id.Pseudo != "xatelitte" || atomic.LoadInt32(&posted) != 1 {
		t.Fatalf("id=%+v posted=%d", id, posted)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/hfr/ -run TestReply -v`
Expected: FAIL — `Reply` signature mismatch (`too many arguments`, no return value).

- [ ] **Step 3: Write minimal implementation**

Add `authenticatedPost` to `client.go`:

```go
// authenticatedPost resolves the current identity, enforces the guard, and
// only then POSTs. Returns the identity used. The whole sequence is atomic.
func (c *Client) authenticatedPost(endpoint string, data url.Values, expect, errCode string) (Identity, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	id, err := c.currentIdentity()
	if err != nil {
		return Identity{}, err
	}
	if err := c.checkIdentity(id, expect); err != nil {
		return Identity{}, err
	}
	doc, err := c.doPost(endpoint, data)
	if err != nil {
		return Identity{}, err
	}
	if err := checkPostSuccess(doc, errCode); err != nil {
		return Identity{}, err
	}
	return id, nil
}
```

Rewrite `Reply` in `post.go` (drop `ensureAuth`/manual `doPost`; keep the form fields):

```go
func (c *Client) Reply(cat, postId int, content, expect string) (Identity, error) {
	data := c.baseFormData(strconv.Itoa(cat), content)
	data.Set("post", strconv.Itoa(postId))
	data.Set("sujet", c.pseudo)
	data.Set("numreponse", "")
	data.Set("numrep", "")
	data.Set("subcat", "")
	data.Set("parents", "")
	data.Set("stickold", "")
	data.Set("cache", "")
	data.Set("search_smilies", "")
	data.Set("ColorUsedMem", "")
	return c.authenticatedPost("/bddpost.php?config=hfr.inc", data, expect, "post")
}
```

Rewrite `CreateTopic` (same pattern, return `(Identity, error)`, end with `return c.authenticatedPost("/bddpost.php?config=hfr.inc", data, expect, "create_topic")`, add `expect string` param, keep its `data.Set(...)` lines, drop `ensureAuth`).

Rewrite `Edit` — the GET stays first (so the re-check inside `authenticatedPost` is the TOCTOU guard right before POST):

```go
func (c *Client) Edit(cat, postId, numreponse int, content, expect string) (Identity, error) {
	editURL := fmt.Sprintf("%s/message.php?config=hfr.inc&cat=%d&post=%d&numreponse=%d",
		c.baseURL, cat, postId, numreponse)
	editDoc, err := c.doGet(editURL)
	if err != nil {
		return Identity{}, fmt.Errorf("edit page fetch failed: %w", err)
	}
	info := parseEditPage(editDoc)

	data := c.baseFormData(strconv.Itoa(cat), content)
	data.Set("post", strconv.Itoa(postId))
	data.Set("numreponse", strconv.Itoa(numreponse))
	data.Set("dest", "")
	data.Set("numrep", "")
	data.Set("parents", "")
	data.Set("stickold", "")
	data.Set("cache", "")
	data.Set("search_smilies", "")
	data.Set("ColorUsedMem", "")
	if info.IsFirstPost {
		data.Set("sujet", info.Subject)
		data.Set("subcat", info.Subcat)
	} else {
		data.Set("sujet", c.pseudo)
		data.Set("subcat", "")
	}
	return c.authenticatedPost("/bdd.php?config=hfr.inc", data, expect, "edit")
}
```

Rewrite `SendMP` in `mp.go`:

```go
func (c *Client) SendMP(dest, subject, content, expect string) (Identity, error) {
	data := c.baseFormData("prive", content)
	data.Set("dest", dest)
	data.Set("sujet", subject)
	data.Set("post", "")
	data.Set("numreponse", "")
	data.Set("numrep", "")
	data.Set("subcat", "")
	data.Set("parents", "")
	data.Set("stickold", "")
	data.Set("cache", "")
	data.Set("search_smilies", "")
	data.Set("ColorUsedMem", "")
	return c.authenticatedPost("/bddpost.php?config=hfr.inc", data, expect, "mp")
}
```

> `ensureAuth` is now unused by writes (login presence is enforced by `currentIdentity`). Leave it defined (still referenced by `FetchQuote` in `reader.go`).

- [ ] **Step 4: Run tests**

Run: `go test ./internal/hfr/ -v && go build ./...`
Expected: hfr package tests PASS. Build of `./cmd/...` will FAIL until Tasks 7–8 update callers — that is expected and fixed there. Confirm the *hfr package test* passes:
`go test ./internal/hfr/ -run TestReply -v` → PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/hfr/client.go internal/hfr/post.go internal/hfr/mp.go internal/hfr/write_guard_test.go
git commit -m "feat(hfr): atomic guarded write path, return Identity (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: Config — expected login + opt-out

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadIdentityEnv(t *testing.T) {
	t.Setenv("HFR_LOGIN", "xatelitte")
	t.Setenv("HFR_PASSWD", "pw")
	t.Setenv("HFR_EXPECT_LOGIN", "xatelitte")
	t.Setenv("HFR_ALLOW_UNGUARDED_WRITES", "1")
	cfg := Load()
	if cfg.ExpectLogin != "xatelitte" {
		t.Fatalf("ExpectLogin = %q", cfg.ExpectLogin)
	}
	if !cfg.AllowUnguarded {
		t.Fatal("AllowUnguarded should be true")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/config/ -run TestLoadIdentityEnv -v`
Expected: FAIL — `cfg.ExpectLogin undefined`.

- [ ] **Step 3: Write minimal implementation**

Extend the `Config` struct and `Load` in `config.go`:

```go
type Config struct {
	Login          string
	Passwd         string
	ExpectLogin    string
	AllowUnguarded bool
}
```

Add `expect_login` to the file parser `switch` in `readFile`:

```go
		case "expect_login":
			cfg.ExpectLogin = v
```

Add env overrides at the end of `Load` (before `return cfg`):

```go
	if v := os.Getenv("HFR_EXPECT_LOGIN"); v != "" {
		cfg.ExpectLogin = v
	}
	if os.Getenv("HFR_ALLOW_UNGUARDED_WRITES") == "1" {
		cfg.AllowUnguarded = true
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/config/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): HFR_EXPECT_LOGIN + HFR_ALLOW_UNGUARDED_WRITES (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: MCP — hfr_whoami, expect fields, wire config

**Files:**
- Modify: `internal/mcp/tools.go` (inputs, handlers, register `hfr_whoami`)
- Modify: `cmd/hfr-mcp/main.go` (inject guard config)

- [ ] **Step 1: Add `expect` to write inputs + a WhoamiInput**

In `tools.go`, add `Expect string \`json:"expect,omitempty" jsonschema:"Compte attendu (pseudo, id:NNNN, ou pseudo:nom) ; l'écriture est refusée si la session ne correspond pas"\`` to `ReplyInput`, `EditInput`, `CreateTopicInput`, `MPInput`. Add:

```go
type WhoamiInput struct{}
```

- [ ] **Step 2: Register the whoami tool**

In `RegisterTools`, add:

```go
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "hfr_whoami",
		Description: "Afficher le compte HFR actif (pseudo + userId) et le compte attendu configuré.",
	}, handleWhoami(client, login))
```

- [ ] **Step 3: Update write handlers + add whoami handler**

Change the four write handlers to pass `input.Expect` and format the returned identity. Example for `handleReply` (apply the same shape to edit/create_topic/mp):

```go
func handleReply(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[ReplyInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ReplyInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		id, err := client.Reply(input.Cat, input.Post, input.Content, input.Expect)
		if err != nil {
			return nil, Result{}, fmt.Errorf("reply failed: %w", err)
		}
		return nil, Result{Message: fmt.Sprintf("Message posté sous %s (userId %s).", id.Pseudo, id.UserID)}, nil
	}
}
```

Add the whoami handler:

```go
func handleWhoami(client *hfr.Client, login LoginFunc) mcp.ToolHandlerFor[WhoamiInput, Result] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input WhoamiInput) (*mcp.CallToolResult, Result, error) {
		if err := login(); err != nil {
			return nil, Result{}, fmt.Errorf("login failed: %w", err)
		}
		id, err := client.Whoami()
		if err != nil {
			return nil, Result{}, fmt.Errorf("whoami failed: %w", err)
		}
		return nil, Result{Message: fmt.Sprintf("Connecté : %s (userId %s).", id.Pseudo, id.UserID)}, nil
	}
}
```

- [ ] **Step 4: Inject guard config in the server**

In `cmd/hfr-mcp/main.go`, after `client := hfr.NewClient()` (line 30):

```go
	client.SetExpectedLogin(cfg.ExpectLogin)
	client.SetAllowUnguarded(cfg.AllowUnguarded)
```

- [ ] **Step 5: Build + commit**

Run: `go build ./... && go vet ./...`
Expected: OK (MCP callers now match new signatures).

```bash
git add internal/mcp/tools.go cmd/hfr-mcp/main.go
git commit -m "feat(mcp): hfr_whoami + expect param + guard wiring (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: CLI — --pseudo flag, whoami subcommand

**Files:**
- Modify: `cmd/hfr/main.go`

- [ ] **Step 1: Parse `--pseudo` and inject guard**

In `main()`, extend flag parsing after the `--auth` block (lines 40–44). Replace that block with:

```go
	args := os.Args[1:]
	auth := false
	pseudo := ""
	for len(args) > 0 {
		switch {
		case args[0] == "--auth":
			auth = true
			args = args[1:]
		case args[0] == "--pseudo" && len(args) > 1:
			pseudo = args[1]
			args = args[2:]
		default:
			goto parsed
		}
	}
parsed:
```

After `client := hfr.NewClient()` and inside the `if auth {` block (after a successful `client.Login`), inject config-based guard plus the flag:

```go
		client.SetExpectedLogin(cfg.ExpectLogin)
		client.SetAllowUnguarded(cfg.AllowUnguarded)
```

(`pseudo` is threaded into the write commands below.) Add `whoami` to the `needsAuth` set by leaving the default (it is not in the read-only list, so `needsAuth` is already true for it).

- [ ] **Step 2: Add the whoami case + thread pseudo into writers**

In the `switch cmd` block add:

```go
	case "whoami":
		id, err := client.Whoami()
		if err != nil {
			die("whoami failed: %v", err)
		}
		fmt.Printf("Connecté : %s (userId %s)\n", id.Pseudo, id.UserID)
```

Update the four write dispatch calls to pass `pseudo`:

```go
	case "new":
		cmdNewTopic(client, args, pseudo)
	case "reply":
		cmdReply(client, args, pseudo)
	case "edit":
		cmdEdit(client, args, pseudo)
	case "mp":
		cmdMP(client, args, pseudo)
```

- [ ] **Step 3: Update the write command functions**

```go
func cmdNewTopic(client *hfr.Client, args []string, expect string) {
	if len(args) < 4 {
		die("usage: hfr new <cat> <subcat> <subject> <content|--file path>")
	}
	cat := mustInt(args[0], "cat")
	subcat := mustInt(args[1], "subcat")
	subject := args[2]
	content := readContent(args[3:])
	id, err := client.CreateTopic(cat, subcat, subject, content, expect)
	if err != nil {
		die("create topic failed: %v", err)
	}
	fmt.Printf("Topic created (as %s / userId %s).\n", id.Pseudo, id.UserID)
}

func cmdReply(client *hfr.Client, args []string, expect string) {
	if len(args) < 3 {
		die("usage: hfr reply <cat> <post> <content|--file path>")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	content := readContent(args[2:])
	id, err := client.Reply(cat, post, content, expect)
	if err != nil {
		die("reply failed: %v", err)
	}
	fmt.Printf("Reply posted (as %s / userId %s).\n", id.Pseudo, id.UserID)
}

func cmdEdit(client *hfr.Client, args []string, expect string) {
	if len(args) < 4 {
		die("usage: hfr edit <cat> <post> <numreponse> <content|--file path>")
	}
	cat := mustInt(args[0], "cat")
	post := mustInt(args[1], "post")
	numreponse := mustInt(args[2], "numreponse")
	content := readContent(args[3:])
	id, err := client.Edit(cat, post, numreponse, content, expect)
	if err != nil {
		die("edit failed: %v", err)
	}
	fmt.Printf("Post edited (as %s / userId %s).\n", id.Pseudo, id.UserID)
}

func cmdMP(client *hfr.Client, args []string, expect string) {
	if len(args) < 3 {
		die("usage: hfr mp <dest> <subject> <content>")
	}
	dest := args[0]
	subject := args[1]
	content := strings.Join(args[2:], " ")
	id, err := client.SendMP(dest, subject, content, expect)
	if err != nil {
		die("mp failed: %v", err)
	}
	fmt.Printf("MP sent (as %s / userId %s).\n", id.Pseudo, id.UserID)
}
```

Update `usage` const: add `whoami` to the command list and `--pseudo <login>` to Options.

- [ ] **Step 4: Build + vet + full test suite**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: build OK, vet OK, all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/hfr/main.go
git commit -m "feat(cli): --pseudo guard + whoami subcommand (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: Docs + version bump

**Files:**
- Modify: `internal/hfr/version.go`
- Modify: `CHANGELOG.md`
- Modify: `README.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Bump version**

`internal/hfr/version.go`: `const Version = "1.2.0"`.

- [ ] **Step 2: CHANGELOG**

Prepend a `## [1.2.0] - 2026-06-14` section: new `hfr_whoami` tool / `hfr whoami` command; `expect` param (MCP) and `--pseudo` flag (CLI); fail-closed identity guard with `HFR_EXPECT_LOGIN` and `HFR_ALLOW_UNGUARDED_WRITES`; write methods now report the account used; login hardening (no partial-auth state).

- [ ] **Step 3: README**

In the Fonctionnalités and Outils MCP tables add `hfr_whoami`. Add `--pseudo` to the CLI options and an Authentification note documenting `HFR_EXPECT_LOGIN` / `HFR_ALLOW_UNGUARDED_WRITES` and the typed `expect` syntax (`pseudo:`, `id:`). (Also fix the pre-existing gap: `hfr_cats` and `hfr_create_topic` are missing from the Outils MCP table.)

- [ ] **Step 4: AGENTS.md**

In "Configuration et authentification (runtime)", document the guard: by default writes require an expected account (`HFR_EXPECT_LOGIN` env / `expect_login=` / `--pseudo` / `expect`); `HFR_ALLOW_UNGUARDED_WRITES=1` to opt out; comparison by pseudo or userId.

- [ ] **Step 5: Final validation + commit**

Run: `go build ./... && go vet ./... && golangci-lint run && go test ./...`
Expected: all green.

```bash
git add internal/hfr/version.go CHANGELOG.md README.md AGENTS.md
git commit -m "docs: document identity guard, bump to v1.2.0 (#32)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

---

## Final live verification (before opening the PR)

- [ ] Build the CLI and confirm against the real forum (read-only + a guarded refusal):

```bash
go build -o /tmp/hfr ./cmd/hfr/
HFR_LOGIN=<agent> HFR_PASSWD=<pw> /tmp/hfr --auth whoami
# expect: "Connecté : <agent> (userId <n>)" with the CORRECT numeric id (cross-check via `hfr quote`)
HFR_LOGIN=<agent> HFR_PASSWD=<pw> HFR_EXPECT_LOGIN=someoneelse /tmp/hfr reply 23 99999 "test"
# expect: refusal, no post
```

- [ ] If the live `userId` is empty or wrong, fix `parseUserID` (Task 4) and its fixture, then re-run Tasks 4–5 tests.
- [ ] Open the PR with `Closes #32`, summarizing the guard, the new tool/flag, and the env vars.

## Self-review notes (coverage vs spec)

- Fail-closed + opt-out → Task 2 (`checkIdentity`) + Task 6 (config).
- Cookie `md_user` as POST authority + atomic path → Task 4 (`currentIdentity`) + Task 5 (`authenticatedPost`).
- Edit TOCTOU re-check before POST → Task 5 (`Edit` GET first, guard inside `authenticatedPost`).
- Typed `expect` (`pseudo:`/`id:`), bare numeric → userId, `"0"` real → Task 1.
- userId fail-closed when unresolved → Task 1 (`identityMatches` error path).
- Login rollback → Task 4. Mutex on shared client → Tasks 4–5. Injectable baseURL → Task 3.
- Methods return Identity → Task 5; surfaced in MCP/CLI → Tasks 7–8.
- `hfr_whoami` / `hfr whoami` → Tasks 7–8. Docs/version → Task 9.
