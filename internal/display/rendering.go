package display

import (
	"log"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/internal/ansi"
	"github.com/giulianopz/newscanoe/internal/cache"
	"github.com/giulianopz/newscanoe/internal/util"
)

func (d *display) renderURLs() {

	cached := make(map[string]*cache.Feed, 0)

	for _, cachedFeed := range d.cache.GetFeeds() {
		cached[cachedFeed.Url] = cachedFeed
	}

	d.rendered = make([][]*cell, 0)
	for row := range d.raw {
		url := d.raw[row]
		if !strings.Contains(string(url), "#") {
			cachedFeed, present := cached[string(url)]
			if !present {
				d.appendToRendered(fromString(string(url)))
			} else {
				if cachedFeed.New {
					d.appendToRendered(fromStringWithStyle(cachedFeed.Title, ansi.BOLD))
				} else {
					d.appendToRendered(fromString(cachedFeed.Title))
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

	d.rendered = make([][]*cell, 0)
	if currentFeed != nil {
		for _, item := range currentFeed.GetItemsOrderedByDate() {
			if item.New {
				d.appendToRendered(fromStringWithStyle(util.RenderArticleRow(item.PubDate, item.Title), ansi.BOLD))
			} else {
				d.appendToRendered(fromString(util.RenderArticleRow(item.PubDate, item.Title)))
			}
		}
	} else {
		d.setTmpBottomMessage(1*time.Second, "cannot load article list!")
	}
}

func (d *display) renderArticleText() {

	log.Default().Println("width: ", d.width)

	var textSpace, margin int = d.width - 1, 0

	if d.width > 100 {
		textSpace = (d.width - 1) / 4 * 2
		margin = ((d.width - 1) - textSpace) / 2
	}

	runes := make([]rune, 0)
	for row := range d.raw {
		if len(d.raw[row]) == 0 {
			runes = append(runes, '\n')
		}
		for _, c := range string(d.raw[row]) {
			runes = append(runes, c)
		}
	}

	d.rendered = make([][]*cell, 0)
	line := make([]*cell, 0)
	for _, c := range runes {

		if c == '\r' || c == '\n' {

			if len(line) != 0 {
				d.rendered = append(d.rendered, add(margin, line))
			}
			d.rendered = append(d.rendered, []*cell{})
			line = make([]*cell, 0)
			continue
		}

		if c == '\t' {
			for i := 0; i < 4; i++ {
				line = append(line, newCell(' '))
			}
			continue
		}

		if len(line) < textSpace {
			line = append(line, newCell(c))
		} else {

			if line[len(line)-1].char != ' ' {

				var lastIdx int = len(line)
				tmp := make([]*cell, 0)
				for i := lastIdx - 1; i >= 0; i-- {
					lastIdx--
					if line[i].char != ' ' {
						tmp = append([]*cell{line[i]}, tmp...)
					} else {
						break
					}
				}

				d.rendered = append(d.rendered, add(margin, line[:lastIdx]))
				line = tmp
			} else {

				d.rendered = append(d.rendered, add(margin, line))
				line = make([]*cell, 0)
			}

			line = append(line, newCell(c))
		}
	}

	if len(line) != 0 {
		d.rendered = append(d.rendered, add(margin, line))
	}
}

func add(margin int, line []*cell) []*cell {
	if margin != 0 {
		padded := make([]*cell, 0)
		for margin != 0 {
			padded = append(padded, newCell(' '))
			margin--
		}
		padded = append(padded, line...)
		return padded
	}
	return line
}
