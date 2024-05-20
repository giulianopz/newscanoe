package html

import (
	"html"
	"io"
	"log"
	"net/http"

	"github.com/giulianopz/go-readability"
)

func ExtractText(url string) (string, error) {

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	reader, err := readability.New(
		string(bs), url,
	)
	if err != nil {
		return "", err
	}
	article, err := reader.Parse()
	if err != nil {
		return "", err
	}

	unescapedText := html.UnescapeString(article.TextContent)
	log.Default().Printf("unescaped article text: %s", unescapedText)
	return unescapedText, nil
}
