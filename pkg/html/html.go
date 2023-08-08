package html

import (
	"html"
	"log"
	"time"

	"github.com/giulianopz/go-readability"
)

func ExtractText(url string) (string, error) {
	article, err := readability.FromURL(url, 3*time.Second)
	if err != nil {
		return "", err
	}
	unescapedText := html.UnescapeString(article.TextContent)
	log.Default().Printf("unescaped article text: %s", unescapedText)
	return unescapedText, nil
}
