package hfr

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// readSinglePage fetches and parses one topic page
func (c *Client) readSinglePage(cat, postId, page int) (*Topic, error) {
	return c.readPage(cat, postId, page, false)
}

// readPage fetches and parses one topic page, optionally in print mode (~1000 posts/page, no signatures)
func (c *Client) readPage(cat, postId, page int, print bool) (*Topic, error) {
	printVal := 0
	if print {
		printVal = 1
	}
	topicURL := fmt.Sprintf("%s/forum2.php?config=hfr.inc&cat=%d&post=%d&page=%d&p=1&sondage=0&owntopic=0&trash=0&trash_post=0&print=%d&numreponse=0&quote_only=0&new=0&nojs=0",
		baseURL, cat, postId, page, printVal)

	doc, err := c.doGet(topicURL)
	if err != nil {
		return nil, fmt.Errorf("read topic failed: %w", err)
	}

	return &Topic{
		Cat:        cat,
		Post:       postId,
		Page:       page,
		TotalPages: parseTotalPages(doc),
		Posts:      parsePosts(doc),
	}, nil
}

// ReadTopicPrint fetches a topic in print mode (~1000 posts/page, no signatures).
// Use last=N to keep only the last N posts (0 = all posts).
func (c *Client) ReadTopicPrint(cat, postId, page, last int) (*Topic, error) {
	if page == 0 {
		// Fetch print page 1 to discover total print pages, then fetch the last
		first, err := c.readPage(cat, postId, 1, true)
		if err != nil {
			return nil, err
		}
		if first.TotalPages <= 1 {
			if last > 0 && len(first.Posts) > last {
				first.Posts = first.Posts[len(first.Posts)-last:]
			}
			return first, nil
		}
		topic, err := c.readPage(cat, postId, first.TotalPages, true)
		if err != nil {
			return nil, err
		}
		topic.TotalPages = first.TotalPages
		if last > 0 && len(topic.Posts) > last {
			topic.Posts = topic.Posts[len(topic.Posts)-last:]
		}
		return topic, nil
	}

	topic, err := c.readPage(cat, postId, page, true)
	if err != nil {
		return nil, err
	}
	if last > 0 && len(topic.Posts) > last {
		topic.Posts = topic.Posts[len(topic.Posts)-last:]
	}
	return topic, nil
}

// ReadTopic fetches a single topic page. Use page=0 for the last page.
func (c *Client) ReadTopic(cat, postId, page int) (*Topic, error) {
	if page == 0 {
		// Fetch page 1 to discover total pages, then fetch the last
		first, err := c.readSinglePage(cat, postId, 1)
		if err != nil {
			return nil, err
		}
		if first.TotalPages <= 1 {
			return first, nil
		}
		topic, err := c.readSinglePage(cat, postId, first.TotalPages)
		if err != nil {
			return nil, err
		}
		topic.TotalPages = first.TotalPages
		return topic, nil
	}
	return c.readSinglePage(cat, postId, page)
}

// ReadTopicRange fetches multiple pages concurrently and returns a single merged Topic.
// Use from=0 to mean "last page", negative values for relative (e.g. from=-9, to=0 = last 10 pages).
func (c *Client) ReadTopicRange(cat, postId, from, to int) (*Topic, error) {
	// Resolve total pages if we need relative/last references
	if from <= 0 || to <= 0 {
		first, err := c.readSinglePage(cat, postId, 1)
		if err != nil {
			return nil, err
		}
		total := first.TotalPages
		if from <= 0 {
			from = total + from
		}
		if to <= 0 {
			to = total + to
		}
		if from < 1 {
			from = 1
		}
		if to < 1 {
			to = 1
		}
	}

	if from > to {
		from, to = to, from
	}

	count := to - from + 1
	results := make([]*Topic, count)
	errs := make([]error, count)

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(idx, page int) {
			defer wg.Done()
			results[idx], errs[idx] = c.readSinglePage(cat, postId, page)
		}(i, from+i)
	}
	wg.Wait()

	// Check errors
	for i, err := range errs {
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", from+i, err)
		}
	}

	// Merge all posts in order
	merged := &Topic{
		Cat:        cat,
		Post:       postId,
		Page:       from,
		TotalPages: results[0].TotalPages,
	}
	for _, t := range results {
		merged.Posts = append(merged.Posts, t.Posts...)
	}

	return merged, nil
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
