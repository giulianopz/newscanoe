package util

import (
	"fmt"
	"time"
)

func RenderArticleRow(pubDate time.Time, title string) string {
	return fmt.Sprintf("%-20s %s", pubDate.Format("2006-January-02"), title)
}
