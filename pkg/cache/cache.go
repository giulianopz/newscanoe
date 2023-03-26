package cache

import (
	"encoding/gob"
	"os"
	"time"

	"github.com/giulianopz/newscanoe/pkg/util"
)

type Cache struct {
	Feeds []*Feed
}

type Feed struct {
	Url   string
	Title string
	Items []*Item
}

type Item struct {
	Title   string
	Url     string
	PubDate time.Time
}

func (c *Cache) Encode() error {

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
	if err := e.Encode(c.Feeds); err != nil {
		return err
	}
	return nil
}

func (c *Cache) Decode() error {

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

	c.Feeds = feeds

	return nil
}
