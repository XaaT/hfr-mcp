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

// parseListTotalPages extracts page count from a category listing (liste_sujet-N.htm links)
func parseListTotalPages(doc *goquery.Document) int {
	max := 1
	doc.Find("td.padding a[href]").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if !strings.Contains(href, "liste_sujet-") {
			return
		}
		// liste_sujet-N.htm
		idx := strings.LastIndex(href, "-")
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

// parseTopicList extracts topics from a forum1.php category listing page
func parseTopicList(doc *goquery.Document) []TopicListItem {
	var topics []TopicListItem

	doc.Find("tr.sujet").Each(func(i int, row *goquery.Selection) {
		item := TopicListItem{}
		item.Sticky = row.HasClass("ligne_sticky")

		// Title + PostID from sujetCase3
		titleLink := row.Find("td.sujetCase3 a.cCatTopic").First()
		if titleLink.Length() == 0 {
			return
		}
		item.Title = strings.TrimSpace(titleLink.Text())

		// PostID from title attribute "Sujet n°XXXXX" or from URL
		if titleAttr, exists := titleLink.Attr("title"); exists {
			if strings.HasPrefix(titleAttr, "Sujet n") {
				// "Sujet n°XXXXX" — extract digits after the last non-digit
				numStr := ""
				for j := len(titleAttr) - 1; j >= 0; j-- {
					if titleAttr[j] >= '0' && titleAttr[j] <= '9' {
						numStr = string(titleAttr[j]) + numStr
					} else if numStr != "" {
						break
					}
				}
				item.PostID, _ = strconv.Atoi(numStr)
			}
		}

		// Author from sujetCase6
		item.Author = strings.TrimSpace(row.Find("td.sujetCase6").Text())

		// Replies from sujetCase7
		repliesStr := strings.TrimSpace(row.Find("td.sujetCase7").Text())
		item.Replies, _ = strconv.Atoi(repliesStr)

		// Views from sujetCase8
		viewsStr := strings.TrimSpace(row.Find("td.sujetCase8").Text())
		item.Views, _ = strconv.Atoi(viewsStr)

		// Last page from sujetCase4 link text
		lastPageStr := strings.TrimSpace(row.Find("td.sujetCase4 a").Text())
		if lastPageStr != "" {
			item.LastPage, _ = strconv.Atoi(lastPageStr)
		} else {
			item.LastPage = 1
		}

		// Last message from sujetCase9
		lastCell := row.Find("td.sujetCase9 a")
		if lastCell.Length() > 0 {
			// Date is text before <b>, author is in <b>
			item.LastAuthor = strings.TrimSpace(lastCell.Find("b").Text())
			fullText := strings.TrimSpace(lastCell.Text())
			// Remove author from end to get date
			if item.LastAuthor != "" {
				idx := strings.LastIndex(fullText, item.LastAuthor)
				if idx > 0 {
					item.LastDate = strings.TrimSpace(fullText[:idx])
				}
			}
			// Clean up nbsp
			item.LastDate = strings.ReplaceAll(item.LastDate, "\u00a0", " ")
		}

		if item.PostID > 0 {
			topics = append(topics, item)
		}
	})

	return topics
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
