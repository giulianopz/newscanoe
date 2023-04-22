package html

import (
	"log"
	"time"

	"github.com/go-shiori/go-readability"
)

func ExtractText(url string) (string, error) {
	article, err := readability.FromURL(url, 2*time.Second)
	if err != nil {
		return "", err
	}
	log.Default().Printf("article text: %s", article.TextContent)
	return article.TextContent, nil
}
