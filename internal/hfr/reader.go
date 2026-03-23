package hfr

import "fmt"

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
