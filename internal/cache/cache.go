package cache

import (
	"encoding/gob"
	"log"
	"os"
	"sync"

	"github.com/giulianopz/newscanoe/internal/config"
	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/util"
)

type Cache struct {
	mu    sync.Mutex
	feeds []*feed.Feed
}

func NewCache() *Cache {
	return &Cache{
		feeds: make([]*feed.Feed, 0),
	}
}

func (c *Cache) GetFeeds() []*feed.Feed {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.feeds
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

func (c *Cache) Decode(filePath string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var feeds []*feed.Feed
	d := gob.NewDecoder(file)
	if err := d.Decode(&feeds); err != nil {
		return err
	}

	c.feeds = feeds

	return nil
}

func (c *Cache) AddFeed(parsedFeed *feed.Feed, url string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cachedFeed := range c.feeds {

		if cachedFeed.Url == url {

			for _, parsedItem := range parsedFeed.Items {

				if !cachedFeed.HasItem(parsedItem.Title) {
					cachedFeed.Items = append(cachedFeed.Items, parsedItem)
				}
			}
			log.Default().Printf("refreshed cached feed with url: %s\n", url)
			return nil
		}
	}

	c.feeds = append(c.feeds, parsedFeed)
	log.Default().Printf("cached a new feed with url: %s\n", url)

	return nil
}

func (c *Cache) Merge(conf *config.Config) {
	m := make(map[string]string)
	for _, f := range conf.Feeds {
		m[f.Url] = f.Alias
	}

	for _, f := range c.feeds {
		f.Alias = m[f.Url]
	}
}
