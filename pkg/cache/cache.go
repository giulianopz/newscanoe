package cache

import (
	"encoding/gob"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
	"golang.org/x/exp/slices"
)

type Cache struct {
	mu    sync.Mutex
	feeds []*Feed
}

func NewCache() *Cache {
	return &Cache{
		feeds: make([]*Feed, 0),
	}

}

func (c *Cache) GetFeeds() []*Feed {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.feeds
}

type Feed struct {
	Url   string
	Title string
	Items []*Item
	New   bool
}

func NewFeed(url, title string) *Feed {
	return &Feed{
		Title: title,
		Url:   url,
		Items: make([]*Item, 0),
		New:   true,
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
	New     bool
}

func NewItem(title, url string, pubDate time.Time) *Item {
	return &Item{
		Title:   title,
		Url:     url,
		PubDate: pubDate,
		New:     true,
	}
}

func (c *Cache) Encode() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	filePath, err := util.GetCacheFilePath()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	e := gob.NewEncoder(file)
	if err := e.Encode(c.feeds); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Decode() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	filePath, err := util.GetCacheFilePath()
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var feeds []*Feed
	d := gob.NewDecoder(file)
	if err := d.Decode(&feeds); err != nil {
		return err
	}

	c.feeds = feeds

	return nil
}

func (c *Cache) AddFeed(parsedFeed *gofeed.Feed, url string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	title := strings.TrimSpace(parsedFeed.Title)

	for _, cachedFeed := range c.feeds {
		if cachedFeed.Url == url {

			cachedFeed.Title = title
			cachedFeed.New = true
			//cachedFeed.Items = make([]*Item, 0)

			for _, parsedItem := range parsedFeed.Items {

				alreadyPresent := slices.ContainsFunc(cachedFeed.Items, func(i *Item) bool {
					return i.Title == parsedItem.Title
				})

				if !alreadyPresent {
					cachedItem := NewItem(parsedItem.Title, parsedItem.Link, *parsedItem.PublishedParsed)
					cachedFeed.Items = append(cachedFeed.Items, cachedItem)
				}
			}
			log.Default().Printf("refreshed cached feed with url: %s\n", url)
			return nil
		}
	}

	newFeed := NewFeed(url, title)
	for _, parsedItem := range parsedFeed.Items {

		pubDate := util.NoPubDate
		if parsedItem.PublishedParsed != nil {
			pubDate = *parsedItem.PublishedParsed
		}
		cachedItem := NewItem(parsedItem.Title, parsedItem.Link, pubDate)
		newFeed.Items = append(newFeed.Items, cachedItem)
	}
	c.feeds = append(c.feeds, newFeed)
	log.Default().Printf("cached a new feed with url: %s\n", url)

	return nil
}

func (c *Cache) AddFeedUrl(url string) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.feeds = append(c.feeds, NewFeed(url, url))
}
