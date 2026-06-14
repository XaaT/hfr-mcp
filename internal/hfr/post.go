package hfr

import (
	"fmt"
	"net/url"
	"strconv"
)

// Reply posts a new message on a topic.
//
// It first GETs the reply form (message.php, like the quote/edit pages) and
// preserves the server-rendered form state — most importantly the email-notify
// checkbox — so the POST mirrors a browser and does not silently unsubscribe the
// user from the topic's email notifications (bug #31). The GET runs BEFORE
// authenticatedPost, exactly like Edit: authenticatedPost re-reads the live
// md_user cookie and re-checks identity immediately before the POST, so the
// fail-closed #32 guard still gates the actual write.
//
// Side effect: this authenticated GET on message.php likely marks the topic as
// read (HFR's "lu" flag), even if the POST is subsequently refused by the guard.
// This is benign — opening the reply form in a browser does the same, and an
// agent replying has typically just read the topic.
func (c *Client) Reply(cat, postId int, content, expect string) (Identity, error) {
	replyURL := fmt.Sprintf("%s/message.php?config=hfr.inc&cat=%d&post=%d&page=0&p=1&new=0",
		c.baseURL, cat, postId)
	preserved := url.Values{}
	if replyDoc, err := c.doGet(replyURL); err == nil {
		preserved = preserveReplyFormFields(replyDoc)
	}
	// If the form fetch fails we fall back to the legacy hardcoded fields rather
	// than block the reply; the notify field is then simply omitted as before.

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

	// Merge preserved server state. Fields we deliberately control above
	// (content_form, identity, sujet, …) are already set and must win, so we only
	// add preserved fields we have not set ourselves — except the notify-style
	// checked controls, which the base form never sets and which carry the
	// subscription state we must round-trip.
	mergePreservedFields(data, preserved)

	return c.authenticatedPost("/bddpost.php?config=hfr.inc", data, expect, "post")
}

// mergePreservedFields copies fields from the server-rendered reply form into
// the outgoing POST without clobbering fields the client deliberately set.
// content_form is never copied (the body is ours). hash_check is never copied:
// the login-time anti-CSRF token in baseFormData is authoritative for the POST.
//
// Deliberately conservative (vs a form-wins allowlist): the client's hardcoded
// fields (signature, sondage, owntopic, …) keep their legacy values rather than
// being round-tripped from the server form. This preserves the pre-#31 behavior
// for those fields (no regression) and limits this change to *adding* the
// subscription/notify state the base form omits — at the cost of not mirroring a
// server-set value on a field we already control. Revisit per-field only if a
// concrete setting is shown to be wrongly overridden.
func mergePreservedFields(data, preserved url.Values) {
	for name, vals := range preserved {
		if name == "content_form" || name == "hash_check" {
			continue
		}
		if _, set := data[name]; set {
			// We already control this field (e.g. cat, post, sujet); keep ours.
			continue
		}
		for _, v := range vals {
			data.Add(name, v)
		}
	}
}

// CreateTopic creates a new topic in a category
func (c *Client) CreateTopic(cat, subcat int, subject, content, expect string) (Identity, error) {
	data := c.baseFormData(strconv.Itoa(cat), content)
	data.Set("post", "")
	data.Set("sujet", subject)
	data.Set("subcat", strconv.Itoa(subcat))
	data.Set("numreponse", "")
	data.Set("numrep", "")
	data.Set("parents", "")
	data.Set("stickold", "")
	data.Set("cache", "cache")
	data.Set("search_smilies", "")
	data.Set("ColorUsedMem", "")
	return c.authenticatedPost("/bddpost.php?config=hfr.inc", data, expect, "create_topic")
}

// Edit modifies an existing post
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
