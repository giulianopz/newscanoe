package feed

import (
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/internal/util"
	"github.com/mmcdole/gofeed"
	"golang.org/x/exp/slices"
)

type FeedFormat string

const (
	RSS  FeedFormat = "rss"
	Atom FeedFormat = "atom"
	JSON FeedFormat = "json"
)

type Feed struct {
	Name        string     `yaml:"name"`
	Url         string     `yaml:"url"`
	Format      FeedFormat `yaml:"format"`
	Alias       string     `yaml:"alias"`
	Items       []*Item    `yaml:"-"`
	UnreadCount int        `yaml:"-"`
}

func NewFeed(name string) *Feed {
	return &Feed{
		Name:  name,
		Items: make([]*Item, 0),
	}
}

func NewFeedFrom(parsedFeed *gofeed.Feed, url string) *Feed {
	title := strings.TrimSpace(parsedFeed.Title)
	f := NewFeed(title).WithUrl(url).WithAlias(title).WithFormat(parsedFeed.FeedType)

	for _, parsedItem := range parsedFeed.Items {
		f.Items = append(f.Items, NewItemFrom(parsedItem))
	}

	return f
}

func (f *Feed) HasItem(title string) bool {
	return slices.ContainsFunc(f.Items, func(i *Item) bool {
		return i.Title == title
	})
}

func (f *Feed) WithUrl(url string) *Feed {
	f.Url = url
	return f
}

func (f *Feed) WithFormat(format string) *Feed {
	f.Format = FeedFormat(strings.ToLower(format))
	return f
}

func (f *Feed) WithAlias(alias string) *Feed {
	f.Alias = alias
	return f
}

func (f *Feed) CountUnread() {
	f.UnreadCount = 0
	for _, i := range f.Items {
		if i.Unread {
			f.UnreadCount++
		}
	}
}

func (f *Feed) GetItemsOrderedByDate() []*Item {

	slices.SortFunc(f.Items, func(a, b *Item) bool {
		if a.PubDate == util.NoPubDate || b.PubDate == util.NoPubDate {
			return strings.Compare(a.Title, b.Title) <= -1
		}
		return a.PubDate.After(b.PubDate)
	})

	return f.Items
}

type Item struct {
	Title   string
	Url     string
	PubDate time.Time
	Unread  bool
}

func NewItem(title, url string, pubDate time.Time) *Item {
	return &Item{
		Title:   title,
		Url:     url,
		PubDate: pubDate,
		Unread:  true,
	}
}

func NewItemFrom(parsedItem *gofeed.Item) *Item {
	pubDate := util.NoPubDate
	if parsedItem.PublishedParsed != nil {
		pubDate = *parsedItem.PublishedParsed
	}
	return NewItem(parsedItem.Title, parsedItem.Link, pubDate)
}
