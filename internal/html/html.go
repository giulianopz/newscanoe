package html

import (
	"io"
	"net/http"

	"github.com/giulianopz/go-readability"

	html2ansi "github.com/giulianopz/go-html2ansi"
)

func ExtractText(url string) ([][]byte, error) {

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	bs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// TODO pass htm2ansi as a serializer
	reader, err := readability.New(
		string(bs), url, readability.LogLevel(-1),
	)
	if err != nil {
		return nil, err
	}
	article, err := reader.Parse()
	if err != nil {
		return nil, err
	}

	res := html2ansi.Convert(article.Content)

	/* 	unescapedText := html.UnescapeString(article.TextContent)
	   	log.Default().Printf("unescaped article text: %s", unescapedText) */

	var txt [][]byte
	for _, b := range res.BlockContent {
		if b.BlockType == html2ansi.TextBlock {
			txt = append(txt, b.Content)
		}
	}
	return txt, nil
}
