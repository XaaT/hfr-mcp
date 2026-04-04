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
