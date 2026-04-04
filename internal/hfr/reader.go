package hfr

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ReadTopic fetches and parses a topic page
func (c *Client) ReadTopic(cat, postId, page int) (*Topic, error) {
	topicURL := fmt.Sprintf("%s/forum2.php?config=hfr.inc&cat=%d&post=%d&page=%d&p=1&sondage=0&owntopic=0&trash=0&trash_post=0&print=0&numreponse=0&quote_only=0&new=0&nojs=0",
		baseURL, cat, postId, page)

	doc, err := c.doGet(topicURL)
	if err != nil {
		return nil, fmt.Errorf("read topic failed: %w", err)
	}

	posts := parsePosts(doc)

	return &Topic{
		Cat:   cat,
		Post:  postId,
		Page:  page,
		Posts: posts,
	}, nil
}

// FetchQuote retrieves the BBCode quote for one or more messages via HFR's message.php reply page.
// For multiple numreponses, it sets the multiquote cookie so HFR returns all quotes in a single request.
func (c *Client) FetchQuote(cat, postId int, numreponses ...int) (string, error) {
	if err := c.ensureAuth(); err != nil {
		return "", err
	}
	if len(numreponses) == 0 {
		return "", &HfrError{Code: "quote", Message: "at least one numreponse required"}
	}

	// Set multiquote cookie: quoteshardwarefr-{cat}-{post}=|num1|num2|...
	u, _ := url.Parse(baseURL)
	cookieVal := ""
	for _, nr := range numreponses {
		cookieVal += fmt.Sprintf("|%d", nr)
	}
	c.http.Jar.SetCookies(u, []*http.Cookie{{
		Name:  fmt.Sprintf("quoteshardwarefr-%d-%d", cat, postId),
		Value: cookieVal,
	}})

	quoteURL := fmt.Sprintf("%s/message.php?config=hfr.inc&cat=%d&post=%d&numrep=%d&page=1&p=1&new=0",
		baseURL, cat, postId, numreponses[0])

	doc, err := c.doGet(quoteURL)
	if err != nil {
		return "", fmt.Errorf("fetch quote failed: %w", err)
	}

	bbcode := strings.TrimSpace(doc.Find("textarea#content_form").Text())
	if bbcode == "" {
		return "", &HfrError{Code: "quote", Message: "quote textarea empty — check cat/post/numreponse params"}
	}

	return bbcode, nil
}
