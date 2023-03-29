package util

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 \.\-]+`)

func RenderTitle(title string) string {
	title = nonAlphaNumericRegex.ReplaceAllString(title, "")
	title = strings.Trim(title, " ")
	return title
}

func RenderArticleRow(pubDate time.Time, title string) string {
	return fmt.Sprintf("%-20s %s", pubDate.Format("2006-January-02"), title)
}
