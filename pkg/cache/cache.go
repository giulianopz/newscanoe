package cache

import (
	"encoding/gob"
	"log"
	"os"
	"time"

	"github.com/giulianopz/newscanoe/pkg/util"
)

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

type Cache struct {
	Feeds []*Feed
}

func (c *Cache) Encode() {

	filePath, err := util.GetCacheFilePath()
	if err != nil {
		//TODO
		panic(err)
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	e := gob.NewEncoder(file)
	if err := e.Encode(c.Feeds); err != nil {
		panic(err)
	}
}

func (c *Cache) Decode() error {

	filePath, err := util.GetCacheFilePath()
	if err != nil {
		//TODO
		panic(err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	var feeds []*Feed
	d := gob.NewDecoder(file)
	if err := d.Decode(&feeds); err != nil {
		panic(err)
	}

	c.Feeds = feeds

	return nil
}
