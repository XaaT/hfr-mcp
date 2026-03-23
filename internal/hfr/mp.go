package hfr

import "strings"

// SendMP sends a private message
func (c *Client) SendMP(dest, subject, content string) error {
	if err := c.ensureAuth(); err != nil {
		return err
	}

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

	doc, err := c.doPost("/bddpost.php?config=hfr.inc", data)
	if err != nil {
		return err
	}

	if respErr := checkResponseErrors(doc); respErr != nil {
		return respErr
	}

	body := doc.Text()
	if !strings.Contains(body, "posté avec succès") {
		return &HfrError{Code: "mp", Message: "MP may have failed: success message not found"}
	}

	return nil
}
