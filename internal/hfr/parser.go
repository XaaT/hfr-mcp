package hfr

import (
	"net/url"
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

// preserveReplyFormFields extracts the server-rendered form state from the
// reply page so a POST approximates what a browser would send. It returns every
// named field of the reply form (the one whose action targets bddpost.php),
// EXCEPT the message body (content_form) and submit/button controls. Checkboxes
// and radios are included only when the page rendered them checked — this is how
// the email-notify subscription is carried: HFR renders the notify checkbox
// "checked" iff the user is currently subscribed, so preserving it verbatim
// keeps the subscription untouched (bug #31). The field name is intentionally
// not hardcoded; whatever HFR names it, a checked checkbox round-trips.
//
// Not a full browser emulation: <select> only emits an explicitly selected
// <option> (no "first option when none selected" fallback, no option-text when
// value is absent). That is sufficient here — the reply form's selects do not
// carry subscription state — but is why this is "approximates", not "mirrors".
func preserveReplyFormFields(doc *goquery.Document) url.Values {
	out := url.Values{}
	form := doc.Find(`form[action*="bddpost.php"]`).First()
	if form.Length() == 0 {
		// Fall back to any form carrying the content_form textarea.
		form = doc.Find("form").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return s.Find("textarea[name=content_form]").Length() > 0
		}).First()
	}
	if form.Length() == 0 {
		return out
	}
	form.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok || name == "" || name == "content_form" {
			return
		}
		typ, _ := s.Attr("type")
		switch strings.ToLower(typ) {
		case "submit", "button", "image", "reset", "file", "password":
			return
		case "checkbox", "radio":
			if _, checked := s.Attr("checked"); !checked {
				return // unchecked controls are not submitted by a browser
			}
			val, has := s.Attr("value")
			if !has || val == "" {
				val = "on" // browser default for a valueless checked box
			}
			out.Add(name, val)
			return
		}
		val, _ := s.Attr("value")
		out.Set(name, val)
	})
	// select: take the selected <option> value of each named select.
	form.Find("select[name]").Each(func(_ int, sel *goquery.Selection) {
		name, _ := sel.Attr("name")
		if name == "" {
			return
		}
		opt := sel.Find("option[selected]").First()
		if opt.Length() == 0 {
			return
		}
		val, _ := opt.Attr("value")
		out.Set(name, val)
	})
	return out
}

// parseNotifyState reads the email-notification subscription state from the
// reply form in doc. It understands two representations used by HFR:
//
//   - checkbox `emaill` (on message.php reply form): checked = subscribed,
//     absent checked attr = not subscribed.
//   - hidden input `emaill` (forum2.php quick-reply): value "1" = subscribed,
//     "0" = not subscribed.
//
// Returns a pointer to true/false, or nil if the field is not found.
func parseNotifyState(doc *goquery.Document) *bool {
	// Locate the reply form (same logic as preserveReplyFormFields).
	form := doc.Find(`form[action*="bddpost.php"]`).First()
	if form.Length() == 0 {
		form = doc.Find("form").FilterFunction(func(_ int, s *goquery.Selection) bool {
			return s.Find("textarea[name=content_form]").Length() > 0
		}).First()
	}
	if form.Length() == 0 {
		return nil
	}

	var result *bool
	form.Find(`input[name="emaill"]`).EachWithBreak(func(_ int, s *goquery.Selection) bool {
		typ, _ := s.Attr("type")
		switch strings.ToLower(typ) {
		case "checkbox":
			_, checked := s.Attr("checked")
			v := checked
			result = &v
			return false // stop
		case "hidden", "":
			val, _ := s.Attr("value")
			v := val == "1"
			result = &v
			return false // stop
		}
		return true
	})
	return result
}

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
