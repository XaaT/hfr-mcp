package hfr

import (
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// parseEditPage extracts FP detection and subcat/subject from an edit page
func parseEditPage(doc *goquery.Document) EditInfo {
	info := EditInfo{}

	sujetInput := doc.Find("input[name=sujet]")
	if sujetInput.Length() == 0 {
		return info
	}

	inputType, _ := sujetInput.Attr("type")
	if strings.ToLower(inputType) != "hidden" {
		// First post: subject is editable, subcat is selectable
		info.IsFirstPost = true
		info.Subject, _ = sujetInput.Attr("value")

		selected := doc.Find("option[selected]")
		if selected.Length() > 0 {
			info.Subcat, _ = selected.Attr("value")
		}
	}

	return info
}

// parsePosts extracts posts from a topic page
func parsePosts(doc *goquery.Document) []Post {
	var posts []Post

	doc.Find("table.messagetable").Each(func(i int, table *goquery.Selection) {
		// Skip ads (no anchor in messCase1)
		anchor := table.Find("td.messCase1 a[name^=t]")
		if anchor.Length() == 0 {
			return
		}

		name, _ := anchor.Attr("name")
		numreponse, _ := strconv.Atoi(strings.TrimPrefix(name, "t"))

		// Author
		author := strings.TrimSpace(table.Find("td.messCase1 b.s2").Text())

		// Date
		dateText := strings.TrimSpace(table.Find("td.messCase2 div.toolbar div.left").Text())
		// Extract just the date part: "Posté le DD-MM-YYYY à HH:MM:SS"
		if idx := strings.Index(dateText, "Posté le "); idx != -1 {
			dateText = strings.TrimSpace(dateText[idx+len("Posté le "):])
			// Clean up nbsp
			dateText = strings.ReplaceAll(dateText, "\u00a0", " ")
		}

		// Content: get text from para div
		paraID := "para" + strconv.Itoa(numreponse)
		content := strings.TrimSpace(doc.Find("#" + paraID).Text())

		posts = append(posts, Post{
			Numreponse: numreponse,
			Author:     author,
			Date:       dateText,
			Content:    content,
		})
	})

	return posts
}
