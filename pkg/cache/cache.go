package cache

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
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
}

func NewFeed(title, url string) *Feed {
	return &Feed{
		Title: title,
		Url:   url,
	}
}

type Item struct {
	Title   string
	Url     string
	PubDate time.Time
}

func NewItem(title, url string, pubDate time.Time) *Item {
	return &Item{
		Title:   title,
		Url:     url,
		PubDate: pubDate,
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

	title := util.RenderTitle(parsedFeed.Title)

	for _, cachedFeed := range c.feeds {
		if cachedFeed.Url == url {

			cachedFeed.Title = title
			cachedFeed.Items = make([]*Item, 0)
			for _, parsedItem := range parsedFeed.Items {
				cachedItem := NewItem(parsedItem.Title, parsedItem.Link, *parsedItem.PublishedParsed)
				cachedFeed.Items = append(cachedFeed.Items, cachedItem)
			}
			log.Default().Printf("cached feed with url: %s\n", url)
			return nil
		}
	}
	return fmt.Errorf("cannot add feed")
}

func (c *Cache) AddFeedUrl(url string) {

	c.mu.Lock()
	defer c.mu.Unlock()

	c.feeds = append(c.feeds, NewFeed(url, url))
}
