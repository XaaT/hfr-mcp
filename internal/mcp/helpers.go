package mcp

import (
	"fmt"
	"strings"

	"github.com/XaaT/hfr-mcp/internal/hfr"
)

// formatTopic converts a Topic to a readable text output
func formatTopic(topic *hfr.Topic) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Topic cat=%d post=%d page=%d (%d posts)\n\n",
		topic.Cat, topic.Post, topic.Page, len(topic.Posts)))

	for _, p := range topic.Posts {
		sb.WriteString(fmt.Sprintf("--- #%d | %s | %s ---\n", p.Numreponse, p.Author, p.Date))
		sb.WriteString(p.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
