package hfr

import (
	"fmt"
	"strconv"
)

// Reply posts a new message on a topic
func (c *Client) Reply(cat, postId int, content string) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

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

	doc, err := c.doPost("/bddpost.php?config=hfr.inc", data)
	if err != nil {
		return err
	}

	return checkPostSuccess(doc, "post")
}

// CreateTopic creates a new topic in a category
func (c *Client) CreateTopic(cat, subcat int, subject, content string) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

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

	doc, err := c.doPost("/bddpost.php?config=hfr.inc", data)
	if err != nil {
		return err
	}

	return checkPostSuccess(doc, "create_topic")
}

// Edit modifies an existing post
func (c *Client) Edit(cat, postId, numreponse int, content string) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

	// Fetch the edit page to detect FP and extract subcat/subject
	editURL := fmt.Sprintf("%s/message.php?config=hfr.inc&cat=%d&post=%d&numreponse=%d",
		baseURL, cat, postId, numreponse)

	editDoc, err := c.doGet(editURL)
	if err != nil {
		return fmt.Errorf("edit page fetch failed: %w", err)
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

	doc, err := c.doPost("/bdd.php?config=hfr.inc", data)
	if err != nil {
		return err
	}

	return checkPostSuccess(doc, "edit")
}
