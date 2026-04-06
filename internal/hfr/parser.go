package hfr

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var (
	reMessageEdited = regexp.MustCompile(`Message édité par .+ le \d{2}-\d{2}-\d{4} à \d{2}:\d{2}:\d{2}`)
	reMessageCited  = regexp.MustCompile(`Message cité \d+ fois?`)
	reSignature     = regexp.MustCompile(`(?m)\s*-{15}\n\t\t\t.*$`)
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

// parseTotalPages extracts the total page count from a topic page
func parseTotalPages(doc *goquery.Document) int {
	// Method 1: hidden input (available when authenticated)
	if val, exists := doc.Find("input[name=page]").Attr("value"); exists {
		if n, err := strconv.Atoi(val); err == nil && n > 1 {
			return n
		}
	}

	// Method 2: find max page number from pagination links
	// HFR pagination uses "Page Suivante" and numbered page links in div.pagepresuiv
	max := 1
	doc.Find("div.pagepresuiv a[href], td a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		// Only match topic pagination: sujet_XXXXX_NNN.htm
		if !strings.Contains(href, "sujet_") {
			return
		}
		idx := strings.LastIndex(href, "_")
		if idx == -1 {
			return
		}
		suffix := href[idx+1:]
		dotIdx := strings.Index(suffix, ".")
		if dotIdx == -1 {
			return
		}
		if n, err := strconv.Atoi(suffix[:dotIdx]); err == nil && n > max {
			max = n
		}
	})

	return max
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

		// Content: get text from para div, clean noise
		paraID := "para" + strconv.Itoa(numreponse)
		content := cleanContent(doc.Find("#" + paraID).Text())

		posts = append(posts, Post{
			Numreponse: numreponse,
			Author:     author,
			Date:       dateText,
			Content:    content,
		})
	})

	return posts
}

// cleanContent strips noise from post content
func cleanContent(s string) string {
	s = reMessageEdited.ReplaceAllString(s, "")
	s = reMessageCited.ReplaceAllString(s, "")
	s = reSignature.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}
