package todoist

import (
	"regexp"
)

var (
	link1 = regexp.MustCompile(`\[((?:[^\[\]]|\[[^\]]*\])*)\]\((https?://\S+)\)`)
	link2 = regexp.MustCompile(`(https?://\S+)\s+\(([^\)]+)\)`)
)

func sanitizeLink(content string) string {
	content = link1.ReplaceAllString(content, "$1")
	content = link2.ReplaceAllString(content, "$2")
	return content
}

func marginLink(content string) string {
	return link1.ReplaceAllString(content, "[$1]( $2 )")
}
