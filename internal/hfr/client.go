package hfr

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const defaultBaseURL = "https://forum.hardware.fr"

// Client handles all interactions with the HFR forum
type Client struct {
	http           *http.Client
	ua             string
	pseudo         string
	hashCheck      string
	authed         bool
	userID         string // resolved numeric HFR user id at login ("" if unknown)
	expectedLogin  string // server-side expected account (HFR_EXPECT_LOGIN)
	allowUnguarded bool   // HFR_ALLOW_UNGUARDED_WRITES opt-out
	baseURL        string // injectable (tests); defaults to defaultBaseURL
	// mu serializes login and the guarded write path (Login / authenticatedPost).
	// checkIdentity is called by the holder of mu and must NOT lock it itself.
	// Setters below are init-only (called before the server starts handling calls).
	mu sync.Mutex
}

// NewClient creates a new HFR client with a cookie jar and timeout
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

// SetExpectedLogin sets the server-side expected account guard.
func (c *Client) SetExpectedLogin(login string) { c.expectedLogin = login }

// SetAllowUnguarded enables writes when no expected account is configured.
func (c *Client) SetAllowUnguarded(b bool) { c.allowUnguarded = b }

// ExpectedLogin returns the configured server-side expected account ("" if none).
func (c *Client) ExpectedLogin() string { return c.expectedLogin }

// GuardActive reports whether writes are guarded. It is inactive only when no
// expected account is set and unguarded writes were explicitly opted into.
func (c *Client) GuardActive() bool { return c.expectedLogin != "" || !c.allowUnguarded }

// Login authenticates with the forum.
// State is mutated only on full success; any failure leaves authed=false.
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
			break
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
// It does not lock c.mu; callers that need atomicity hold it.
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

// ensureAuth checks that the client is authenticated
func (c *Client) ensureAuth() error {
	if !c.authed {
		return ErrNotAuthenticated
	}
	return nil
}

// baseFormData returns the common form fields for posting.
// It reads c.pseudo/c.hashCheck without holding c.mu; this is safe because
// they are written once at login, before any write call (lazy-login model).
func (c *Client) baseFormData(cat string, content string) url.Values {
	return url.Values{
		"hash_check":   {c.hashCheck},
		"cat":          {cat},
		"content_form": {content},
		"pseudo":       {c.pseudo},
		"password":     {""},
		"verifrequet":  {"1100"},
		"MsgIcon":      {"1"},
		"signature":    {"1"},
		"wysiwyg":      {"0"},
		"new":          {"0"},
		"page":         {"1"},
		"p":            {"1"},
		"sondage":      {"0"},
		"sond":         {"0"},
		"owntopic":     {"0"},
		"config":       {"hfr.inc"},
		"submit":       {"Valider+votre+message"},
	}
}

// doPost sends a POST request and returns the parsed document
func (c *Client) doPost(endpoint string, data url.Values) (*goquery.Document, error) {
	body := strings.NewReader(data.Encode())
	req, err := http.NewRequest("POST", c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("post request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.ua)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("post request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	return doc, nil
}

// doGet sends a GET request and returns the parsed document
func (c *Client) doGet(fullURL string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	req.Header.Set("User-Agent", c.ua)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response parse failed: %w", err)
	}

	return doc, nil
}

// authenticatedPost resolves the current identity, enforces the guard, and
// only then POSTs. Returns the identity used. The identity check and the POST
// are atomic; any pre-fetch GET (e.g. Edit's edit-page load) is intentionally
// outside this lock — the guard still re-checks immediately before the POST.
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

// checkPostSuccess validates that a POST was successful by checking for errors then the success marker
func checkPostSuccess(doc *goquery.Document, errCode string) error {
	if respErr := checkResponseErrors(doc); respErr != nil {
		return respErr
	}

	body := doc.Text()
	// HFR uses "posté avec succès" (MP) or "postée avec succès" (reply) or "édité avec succès" (edit)
	if !strings.Contains(body, "avec succès") {
		// Truncate body for error context
		preview := strings.TrimSpace(body)
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		return &HfrError{Code: errCode, Message: errCode + " may have failed: " + preview}
	}

	return nil
}

// checkResponseErrors parses common HFR error messages from a response
func checkResponseErrors(doc *goquery.Document) error {
	body := doc.Text()

	errors := map[string]*HfrError{
		"Vous n'avez pas les droits pour":                    ErrNoRights,
		"Afin de prevenir les tentatives de flood":           ErrFloodLimit,
		"Afin de prévenir les tentatives de flood":           ErrFloodLimit,
		"Ce sujet est fermé":                                 ErrTopicLocked,
		"Vous devez être identifié":                          ErrSessionExpired,
		"Vous devez remplir tous les champs avant de poster": {Code: "post", Message: "content or subject missing"},
		"Vous devez entrez un destinataire":                  {Code: "post", Message: "recipient missing"},
	}

	for msg, hfrErr := range errors {
		if strings.Contains(body, msg) {
			return hfrErr
		}
	}

	return nil
}
