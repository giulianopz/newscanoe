package display

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/pkg/ansi"
	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/util"
)

func (d *display) renderURLs() {

	cached := make(map[string]*cache.Feed, 0)

	for _, cachedFeed := range d.cache.GetFeeds() {
		cached[cachedFeed.Url] = cachedFeed
	}

	d.rendered = make([][]byte, 0)
	for row := range d.raw {
		url := d.raw[row]
		if !strings.Contains(string(url), "#") {
			cachedFeed, present := cached[string(url)]
			if !present {
				d.appendToRendered(string(url))
			} else {
				if cachedFeed.New {
					d.appendToRendered(fmt.Sprintf("%s%s%s", ansi.SGR(ansi.BOLD), cachedFeed.Title, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF)))
				} else {
					d.appendToRendered(cachedFeed.Title)
				}
			}
		}
	}
}

func (d *display) renderArticlesList() {

	var currentFeed *cache.Feed
	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == d.currentFeedUrl {
			currentFeed = cachedFeed
		}
	}

	d.rendered = make([][]byte, 0)
	if currentFeed != nil {
		for _, item := range currentFeed.GetItemsOrderedByDate() {
			if item.New {
				d.appendToRendered(fmt.Sprintf("%s%s%s", ansi.SGR(ansi.BOLD), util.RenderArticleRow(item.PubDate, item.Title), ansi.SGR(ansi.ALL_ATTRIBUTES_OFF)))
			} else {
				d.appendToRendered(util.RenderArticleRow(item.PubDate, item.Title))
			}
		}
	} else {
		d.setTmpBottomMessage(1*time.Second, "cannot load article list!")
	}

}

func (d *display) renderArticleText() {

	log.Default().Println("width: ", d.width)

	chars := make([]byte, 0)
	for row := range d.raw {
		if len(d.raw[row]) == 0 {
			chars = append(chars, '\n')
		}
		for _, c := range d.raw[row] {
			chars = append(chars, c)
		}
	}

	d.rendered = make([][]byte, 0)
	line := make([]byte, 0)
	for _, c := range chars {

		if c == '\r' || c == '\n' {

			if len(line) != 0 {
				d.rendered = append(d.rendered, line)
			}
			d.rendered = append(d.rendered, []byte{})
			line = make([]byte, 0)
			continue
		}

		if c == '\t' {
			for i := 0; i < 4; i++ {
				line = append(line, ' ')
			}
			continue
		}

		if len(line) < d.width-1 {
			line = append(line, c)
		} else {
			d.rendered = append(d.rendered, line)
			line = make([]byte, 0)
			line = append(line, c)
		}
	}

	if len(line) != 0 {
		d.rendered = append(d.rendered, line)
	}
}
