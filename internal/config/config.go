package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/util"
)

type Config struct {
	mu    sync.Mutex
	Feeds []*feed.Feed
}

func (c *Config) Encode() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	filePath, err := util.GetConfigFilePath()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		slog.Error("cannot open config file", "err", err)
		return err
	}
	defer f.Close()

	for _, feed := range c.Feeds {

		_, err := fmt.Fprintf(f, "%s #%q\n", feed.Url, feed.Name)
		if err != nil {
			slog.Error("cannot write to config file", "err", err)
			return err
		}
	}

	if err := f.Sync(); err != nil {
		slog.Error("cannot sync config file", "err", err)
		return err
	}

	return nil
}

func (c *Config) Decode(filePath string) error {

	c.mu.Lock()
	defer c.mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}

	c.Feeds = make([]*feed.Feed, 0)

	s := bufio.NewScanner(f)
	for s.Scan() {

		f := &feed.Feed{}
		_, err := fmt.Sscanf(s.Text(), "%s #%q", &f.Url, &f.Name)
		if err != nil {
			return err
		}
		c.Feeds = append(c.Feeds, f)
	}

	return nil
}

func (c *Config) AddFeed(parsedFeed *feed.Feed, url string) error {
	for _, f := range c.Feeds {
		if f.Url == url {
			return fmt.Errorf("already present in config: %q", url)
		}
	}
	c.Feeds = append(c.Feeds, parsedFeed)
	return nil
}
