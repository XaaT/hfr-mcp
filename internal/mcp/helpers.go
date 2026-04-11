package mcp

import (
	"fmt"
	"strings"

	"github.com/XaaT/hfr-mcp/internal/hfr"
)

// formatTopic converts a Topic to a readable text output
func formatTopic(topic *hfr.Topic) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Topic cat=%d post=%d page=%d/%d (%d posts)\n\n",
		topic.Cat, topic.Post, topic.Page, topic.TotalPages, len(topic.Posts))

	for _, p := range topic.Posts {
		fmt.Fprintf(&sb, "--- #%d | %s | %s ---\n", p.Numreponse, p.Author, p.Date)
		sb.WriteString(p.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// formatTopicList converts a TopicList to a readable text output
func formatTopicList(list *hfr.TopicList) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Topics cat=%d subcat=%d page=%d/%d (%d topics)\n\n",
		list.Cat, list.Subcat, list.Page, list.TotalPages, len(list.Topics))

	for _, t := range list.Topics {
		sticky := ""
		if t.Sticky {
			sticky = " [sticky]"
		}
		fmt.Fprintf(&sb, "--- post=%d | %s%s ---\n", t.PostID, t.Title, sticky)
		fmt.Fprintf(&sb, "  by %s | %d replies | %d views | %d pages\n", t.Author, t.Replies, t.Views, t.LastPage)
		fmt.Fprintf(&sb, "  last: %s by %s\n\n", t.LastDate, t.LastAuthor)
	}

	return sb.String()
}
