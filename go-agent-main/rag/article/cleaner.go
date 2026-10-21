package article

import (
	"html"
	"regexp"
	"strings"
)

var (
	imagePattern        = regexp.MustCompile(`!\[([^\]]*)\]\([^)]*\)`)
	linkPattern         = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	htmlTagPattern      = regexp.MustCompile(`<[^>]+>`)
	markdownMarkPattern = regexp.MustCompile(`(?m)^\s{0,3}(#{1,6}\s+|>\s?|[-*+]\s+|\d+[.)]\s+)`)
	blankLinesPattern   = regexp.MustCompile(`\n{3,}`)
)

func CleanMarkdown(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "```", "")
	content = imagePattern.ReplaceAllString(content, "$1")
	content = linkPattern.ReplaceAllString(content, "$1")
	content = htmlTagPattern.ReplaceAllString(content, " ")
	content = markdownMarkPattern.ReplaceAllString(content, "")
	content = html.UnescapeString(content)
	content = blankLinesPattern.ReplaceAllString(content, "\n\n")
	return strings.TrimSpace(content)
}
