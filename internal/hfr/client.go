package hfr

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const baseURL = "https://forum.hardware.fr"

// Client handles all interactions with the HFR forum
type Client struct {
	http      *http.Client
	ua        string
	pseudo    string
	hashCheck string
	authed    bool
}

// NewClient creates a new HFR client with a cookie jar and timeout
func NewClient() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		http: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
		ua: "hfr-mcp/" + Version,
	}
}

// Login authenticates with the forum
func (c *Client) Login(pseudo, password string) error {
	data := url.Values{
		"pseudo":   {pseudo},
		"password": {password},
	}

	doc, err := c.doPost("/login_validation.php?config=hfr.inc", data)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	body := doc.Text()

	if strings.Contains(body, "Votre mot de passe ou nom d'utilisateur n'est pas valide") {
		return ErrInvalidCreds
	}

	// Check cookie
	u, _ := url.Parse(baseURL)
	for _, cookie := range c.http.Jar.Cookies(u) {
		if cookie.Name == "md_user" && cookie.Value == pseudo {
			c.pseudo = pseudo
			c.authed = true
			return c.fetchHashCheck()
		}
	}

	return &HfrError{Code: "auth", Message: "login failed: md_user cookie not set"}
}

// fetchHashCheck retrieves the anti-CSRF token
func (c *Client) fetchHashCheck() error {
	doc, err := c.doGet(baseURL + "/user/editprofil.php?config=hardwarefr.inc")
	if err != nil {
		return fmt.Errorf("hash_check failed: %w", err)
	}

	hash, exists := doc.Find("input[name=hash_check]").Attr("value")
	if !exists || hash == "" {
		return ErrNoHashCheck
	}

	c.hashCheck = hash
	return nil
}

// ensureAuth checks that the client is authenticated
func (c *Client) ensureAuth() error {
	if !c.authed {
		return ErrNotAuthenticated
	}
	return nil
}

// baseFormData returns the common form fields for posting
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
	req, err := http.NewRequest("POST", baseURL+endpoint, body)
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
		"Vous n'avez pas les droits pour":                          ErrNoRights,
		"Afin de prevenir les tentatives de flood":                 ErrFloodLimit,
		"Afin de prévenir les tentatives de flood":                 ErrFloodLimit,
		"Ce sujet est fermé":                                       ErrTopicLocked,
		"Vous devez être identifié":                                ErrSessionExpired,
		"Vous devez remplir tous les champs avant de poster":       {Code: "post", Message: "content or subject missing"},
		"Vous devez entrez un destinataire":                        {Code: "post", Message: "recipient missing"},
	}

	for msg, hfrErr := range errors {
		if strings.Contains(body, msg) {
			return hfrErr
		}
	}

	return nil
}
