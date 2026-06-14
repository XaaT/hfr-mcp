package hfr

import (
	"fmt"
	"strconv"
)

// Reply posts a new message on a topic
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
