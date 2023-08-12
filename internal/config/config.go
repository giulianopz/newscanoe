package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/util"
	"gopkg.in/yaml.v3"
)

type Config struct {
	mu    sync.Mutex
	Feeds []*feed.Feed `yaml:"feeds"`
}

func (c *Config) Encode() error {

	c.mu.Lock()
	defer c.mu.Unlock()

	filePath, err := util.GetConfigFilePath()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer file.Close()

	e := yaml.NewEncoder(file)
	if err := e.Encode(c); err != nil {
		return err
	}

	if err := e.Close(); err != nil {
		return err
	}

	return nil
}

func (c *Config) Decode(filePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	bs, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(bs, c); err != nil {
		return err
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
