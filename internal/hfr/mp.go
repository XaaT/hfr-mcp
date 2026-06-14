package hfr

// SendMP sends a private message
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
