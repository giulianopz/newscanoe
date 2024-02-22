package feed

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/atom"
	"github.com/mmcdole/gofeed/json"
	"github.com/mmcdole/gofeed/rss"
)

type Parser struct {
	translator *translator
	*gofeed.Parser
}

func NewParser() *Parser {
	t := &translator{}

	p := gofeed.NewParser()
	p.RSSTranslator = t
	p.AtomTranslator = t
	p.JSONTranslator = t

	return &Parser{
		translator: t,
		Parser:     p,
	}
}

func (p *Parser) Parse(url string) (*Feed, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	parsedFeed, err := p.ParseURLWithContext(url, ctx)
	if err != nil {
		slog.Error("cannot parse feed", "url", url, "err", err)
		return nil, err
	}
	return NewFeedFrom(parsedFeed, url), nil
}

type translator struct{}

func (t *translator) Translate(feed interface{}) (*gofeed.Feed, error) {

	result := &gofeed.Feed{}
	switch f := feed.(type) {
	case *rss.Feed:
		{
			result.FeedType = "rss"
			result.Title = t.rssFeedTitle(f)
			result.Link = t.rssFeedLink(f)
			result.Items = t.rssFeedItems(f)
		}
	case *atom.Feed:
		{
			result.FeedType = "atom"
			result.Title = f.Title
			result.Link = t.atomFeedLink(f)
			result.Items = t.atomFeedItems(f)
		}

	case *json.Feed:
		{
			result.FeedType = "json"
			result.Title = f.Title
			result.Link = f.HomePageURL
			result.Items = t.jsonFeedItems(f)
		}
	default:
		return nil, fmt.Errorf("cannot translate feed of type: %T", feed)
	}
	return result, nil
}

// RSS

func (t *translator) rssFeedLink(rss *rss.Feed) (link string) {
	if rss.Link != "" {
		link = rss.Link
	} else if rss.ITunesExt != nil && rss.ITunesExt.Subtitle != "" {
		link = rss.ITunesExt.Subtitle
	}
	return
}

func (t *translator) rssFeedTitle(rss *rss.Feed) string {
	var title string
	if rss.Title != "" {
		title = rss.Title
	} else if rss.DublinCoreExt != nil && rss.DublinCoreExt.Title != nil {
		title = firstEntry(rss.DublinCoreExt.Title)
	}
	return title
}

func (t *translator) rssFeedItems(rss *rss.Feed) []*gofeed.Item {
	items := []*gofeed.Item{}
	for _, i := range rss.Items {
		items = append(items, t.rssFeedItem(i))
	}
	return items
}

func (t *translator) rssFeedItem(rssItem *rss.Item) *gofeed.Item {
	item := &gofeed.Item{}
	item.Title = t.rssItemTitle(rssItem)
	item.Link = rssItem.Link
	item.PublishedParsed = t.rssItemPublishedParsed(rssItem)
	return item
}

func (t *translator) rssItemTitle(rssItem *rss.Item) (title string) {
	if rssItem.Title != "" {
		title = rssItem.Title
	} else if rssItem.DublinCoreExt != nil && rssItem.DublinCoreExt.Title != nil {
		title = firstEntry(rssItem.DublinCoreExt.Title)
	}
	return
}

func (t *translator) rssItemPublishedParsed(rssItem *rss.Item) (pubDate *time.Time) {
	if rssItem.PubDateParsed != nil {
		return rssItem.PubDateParsed
	} else if rssItem.DublinCoreExt != nil && rssItem.DublinCoreExt.Date != nil {
		pubDateText := firstEntry(rssItem.DublinCoreExt.Date)
		pubDateParsed, err := parseDate(pubDateText)
		if err == nil {
			pubDate = pubDateParsed
		}
	}
	return
}

// Atom

func (t *translator) atomFeedLink(atom *atom.Feed) (link string) {
	l := firstLinkWithType("alternate", atom.Links)
	if l != nil {
		link = l.Href
	}
	return
}

func firstLinkWithType(linkType string, links []*atom.Link) *atom.Link {
	if links == nil {
		return nil
	}

	for _, link := range links {
		if link.Rel == linkType {
			return link
		}
	}
	return nil
}

func (t *translator) atomFeedItems(atom *atom.Feed) []*gofeed.Item {
	items := []*gofeed.Item{}
	for _, entry := range atom.Entries {
		items = append(items, t.atomFeedItem(entry))
	}
	return items
}

func (t *translator) atomFeedItem(entry *atom.Entry) *gofeed.Item {
	item := &gofeed.Item{}
	item.Title = entry.Title
	item.Link = t.atomItemLink(entry)
	item.PublishedParsed = t.atomItemPublishedParsed(entry)
	return item
}

func (t *translator) atomItemLink(entry *atom.Entry) string {
	if l := firstLinkWithType("alternate", entry.Links); l != nil {
		return l.Href
	}
	return ""
}

func (t *translator) atomItemPublishedParsed(entry *atom.Entry) *time.Time {
	published := entry.PublishedParsed
	if published == nil {
		published = entry.UpdatedParsed
	}
	return published
}

// JSON

func (t *translator) jsonFeedItems(json *json.Feed) []*gofeed.Item {
	items := []*gofeed.Item{}
	for _, i := range json.Items {
		items = append(items, t.jsonFeedItem(i))
	}
	return items
}

func (t *translator) jsonFeedItem(jsonItem *json.Item) *gofeed.Item {
	item := &gofeed.Item{}
	item.Link = jsonItem.URL
	item.Title = jsonItem.Title
	item.PublishedParsed = t.jsonItemPublishedParsed(jsonItem)
	return item
}

func (t *translator) jsonItemPublishedParsed(jsonItem *json.Item) *time.Time {
	if jsonItem.DatePublished != "" {
		publishTime, err := parseDate(jsonItem.DatePublished)
		if err != nil {
			log.Default().Println("cannot parse date:", jsonItem.DatePublished)
		} else {
			return publishTime
		}
	}
	return nil
}

// common

func firstEntry(entries []string) string {
	if entries != nil {
		return ""
	}
	return entries[0]
}

var dateFormats = []string{
	time.RFC822,  // RSS
	time.RFC822Z, // RSS
	time.RFC3339, // Atom
	time.UnixDate,
	time.RubyDate,
	time.RFC850,
	time.RFC1123Z,
	time.RFC1123,
	time.ANSIC,
	"Mon, January 2 2006 15:04:05 -0700",
	"Mon, Jan 2 2006 15:04:05 -700",
	"Mon, Jan 2 2006 15:04:05 -0700",
	"Mon Jan 2 15:04 2006",
	"Mon Jan 02, 2006 3:04 pm",
	"Mon Jan 02 2006 15:04:05 -0700",
	"Mon Jan 02 2006 15:04:05 GMT-0700 (MST)",
	"Monday, January 2, 2006 03:04 PM",
	"Monday, January 2, 2006",
	"Monday, January 02, 2006",
	"Monday, 2 January 2006 15:04:05 -0700",
	"Monday, 2 Jan 2006 15:04:05 -0700",
	"Monday, 02 January 2006 15:04:05 -0700",
	"Monday, 02 January 2006 15:04:05",
	"Mon, 2 January 2006, 15:04 -0700",
	"Mon, 2 January 2006 15:04:05 -0700",
	"Mon, 2 January 2006",
	"Mon, 2 Jan 2006 3:04:05 PM -0700",
	"Mon, 2 Jan 2006 15:4:5 -0700 GMT",
	"Mon, 2, Jan 2006 15:4",
	"Mon, 2 Jan 2006, 15:04 -0700",
	"Mon, 2 Jan 2006 15:04 -0700",
	"Mon, 2 Jan 2006 15:04:05 UT",
	"Mon, 2 Jan 2006 15:04:05 -0700 MST",
	"Mon, 2 Jan 2006 15:04:05-0700",
	"Mon, 2 Jan 2006 15:04:05-07:00",
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"Mon, 2 Jan 2006 15:04:05",
	"Mon, 2 Jan 2006 15:04",
	"Mon,2 Jan 2006",
	"Mon, 2 Jan 2006",
	"Mon, 2 Jan 06 15:04:05 -0700",
	"Mon, 2006-01-02 15:04",
	"Mon, 02 January 2006",
	"Mon, 02 Jan 2006 15 -0700",
	"Mon, 02 Jan 2006 15:04 -0700",
	"Mon, 02 Jan 2006 15:04:05 Z",
	"Mon, 02 Jan 2006 15:04:05 UT",
	"Mon, 02 Jan 2006 15:04:05 MST-07:00",
	"Mon, 02 Jan 2006 15:04:05 MST -0700",
	"Mon, 02 Jan 2006 15:04:05 GMT-0700",
	"Mon,02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 -07:00",
	"Mon, 02 Jan 2006 15:04:05 --0700",
	"Mon 02 Jan 2006 15:04:05 -0700",
	"Mon, 02 Jan 2006 15:04:05 -07",
	"Mon, 02 Jan 2006 15:04:05 00",
	"Mon, 02 Jan 2006 15:04:05",
	"Mon, 02 Jan 2006 15:4:5 Z",
	"Mon, 02 Jan 2006",
	"January 2, 2006 3:04 PM",
	"January 2, 2006, 3:04 p.m.",
	"January 2, 2006 15:04:05",
	"January 2, 2006 03:04 PM",
	"January 2, 2006",
	"January 02, 2006 15:04",
	"January 02, 2006 03:04 PM",
	"January 02, 2006",
	"Jan 2, 2006 3:04:05 PM",
	"Jan 2, 2006",
	"Jan 02 2006 03:04:05PM",
	"Jan 02, 2006",
	"6/1/2 15:04",
	"6-1-2 15:04",
	"2 January 2006 15:04:05 -0700",
	"2 January 2006",
	"2 Jan 2006 15:04:05 Z",
	"2 Jan 2006 15:04:05 -0700",
	"2 Jan 2006",
	"2.1.2006 15:04:05",
	"2/1/2006",
	"2-1-2006",
	"2006 January 02",
	"2006-1-2T15:04:05Z",
	"2006-1-2 15:04:05",
	"2006-1-2",
	"2006-1-02T15:04:05Z",
	"2006-01-02T15:04Z",
	"2006-01-02T15:04-07:00",
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00:00",
	"2006-01-02T15:04:05:-0700",
	"2006-01-02T15:04:05-0700",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05 -0700",
	"2006-01-02T15:04:05:00",
	"2006-01-02T15:04:05",
	"2006-01-02 at 15:04:05",
	"2006-01-02 15:04:05Z",
	"2006-01-02 15:04:05-0700",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02 15:04",
	"2006-01-02 00:00:00.0 15:04:05.0 -0700",
	"2006/01/02",
	"2006-01-02",
	"15:04 02.01.2006 -0700",
	"1/2/2006 3:04:05 PM",
	"1/2/2006",
	"06/1/2 15:04",
	"06-1-2 15:04",
	"02 Monday, Jan 2006 15:04",
	"02 Jan 2006 15:04:05 UT",
	"02 Jan 2006 15:04:05 -0700",
	"02 Jan 2006 15:04:05",
	"02 Jan 2006",
	"02.01.2006 15:04:05",
	"02/01/2006 15:04:05",
	"02.01.2006 15:04",
	"02/01/2006 - 15:04",
	"02.01.2006 -0700",
	"02/01/2006",
	"02-01-2006",
	"01/02/2006 3:04 PM",
	"01/02/2006 - 15:04",
	"01/02/2006",
	"01-02-2006",
}

var dateFormatsWithNamedZone = []string{
	"Mon, January 02, 2006, 15:04:05 MST",
	"Mon, January 02, 2006 15:04:05 MST",
	"Mon, Jan 2, 2006 15:04 MST",
	"Mon, Jan 2 2006 15:04 MST",
	"Mon, Jan 2, 2006 15:04:05 MST",
	"Mon Jan 2 15:04:05 2006 MST",
	"Mon, Jan 02,2006 15:04:05 MST",
	"Monday, January 2, 2006 15:04:05 MST",
	"Monday, 2 January 2006 15:04:05 MST",
	"Monday, 2 Jan 2006 15:04:05 MST",
	"Monday, 02 January 2006 15:04:05 MST",
	"Mon, 2 January 2006 15:04 MST",
	"Mon, 2 January 2006, 15:04:05 MST",
	"Mon, 2 January 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:4:5 MST",
	"Mon, 2 Jan 2006 15:04 MST",
	"Mon, 2 Jan 2006 15:04:05MST",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"Mon 2 Jan 2006 15:04:05 MST",
	"mon,2 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 15:04:05 MST",
	"Mon, 2 Jan 06 15:04:05 MST",
	"Mon,02 January 2006 14:04:05 MST",
	"Mon, 02 Jan 2006 3:04:05 PM MST",
	"Mon,02 Jan 2006 15:04 MST",
	"Mon, 02 Jan 2006 15:04 MST",
	"Mon, 02 Jan 2006, 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05MST",
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon , 02 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 06 15:04:05 MST",
	"January 2, 2006 15:04:05 MST",
	"January 02, 2006 15:04:05 MST",
	"Jan 2, 2006 3:04:05 PM MST",
	"Jan 2, 2006 15:04:05 MST",
	"2 January 2006 15:04:05 MST",
	"2 Jan 2006 15:04:05 MST",
	"2006-01-02 15:04:05 MST",
	"1/2/2006 3:04:05 PM MST",
	"1/2/2006 15:04:05 MST",
	"02 Jan 2006 15:04 MST",
	"02 Jan 2006 15:04:05 MST",
	"02/01/2006 15:04 MST",
	"02-01-2006 15:04:05 MST",
	"01/02/2006 15:04:05 MST",
}

func parseDate(ds string) (*time.Time, error) {
	d := strings.TrimSpace(ds)
	if d == "" {
		return nil, fmt.Errorf("date string is empty")
	}
	for _, f := range dateFormats {
		if t, err := time.Parse(f, d); err == nil {
			return &t, nil
		}
	}
	for _, f := range dateFormatsWithNamedZone {
		t, err := time.Parse(f, d)
		if err != nil {
			continue
		}

		loc, err := time.LoadLocation(t.Location().String())
		if err != nil {
			return &t, nil
		}

		if t, err := time.ParseInLocation(f, ds, loc); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("failed to parse date: %s", ds)
}
