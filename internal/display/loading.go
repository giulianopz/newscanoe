package display

import (
	"bufio"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/html"
	"github.com/giulianopz/newscanoe/internal/util"
	"golang.org/x/sync/errgroup"
)

func (d *display) LoadFeedList() error {

	d.mu.Lock()
	defer d.mu.Unlock()

	d.resetRows()

	if len(d.config.Feeds) == 0 {
		d.setBottomMessage("no feed url: type 'a' to add one now")
	} else {

		sort.SliceStable(d.config.Feeds, func(i, j int) bool {
			return strings.ToLower(d.config.Feeds[i].Name) < strings.ToLower(d.config.Feeds[j].Name)
		})
		for _, f := range d.config.Feeds {
			d.appendToRaw(f.Url)
		}
		d.setBottomMessage(urlsListSectionMsg)
	}

	d.renderFeedList()

	d.setTopMessage("")

	d.current.cy = 1
	d.current.cx = 1
	d.currentSection = URLS_LIST
	return nil
}

func (d *display) fetchFeed(url string) (*feed.Feed, error) {

	d.mu.Lock()
	defer d.mu.Unlock()

	parsedFeed, err := d.parser.Parse(url)
	if err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, "cannot parse feed!")
		return nil, err
	}

	if err := d.cache.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
		return nil, err
	}

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()
	return parsedFeed, nil
}

func (d *display) fetchAllFeeds() {

	start := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	origMsg := d.bottomBarMsg
	d.setBottomMessage("loading all feeds, please wait...")
	d.RefreshScreen()

	g := new(errgroup.Group)
	g.SetLimit(-1)

	for id := range d.raw {

		id := id

		g.Go(func() error {
			url := string(d.raw[id])

			log.Default().Printf("loading feed #%d from url %s\n", id, url)

			parsedFeed, err := d.parser.Parse(url)
			if err != nil {
				log.Default().Println(err)
				return err
			}

			if err := d.cache.AddFeed(parsedFeed, url); err != nil {
				log.Default().Println(err)
				return err
			}
			return nil
		})

	}
	d.setBottomMessage(origMsg)

	if err := g.Wait(); err != nil {
		d.setTmpBottomMessage(3*time.Second, "cannot reload all feeds!")
	}

	d.RefreshScreen()

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()

	log.Default().Println("reloaded all feeds in: ", time.Since(start))
}

func (d *display) loadArticleList(url string) error {

	d.mu.Lock()
	defer d.mu.Unlock()

	var found bool
	for _, cachedFeed := range d.cache.GetFeeds() {

		if cachedFeed.Url == url {

			found = true

			if len(cachedFeed.Items) == 0 {
				d.setTmpBottomMessage(3*time.Second, "feed not yet loaded: press r!")
				return fmt.Errorf("feed not loaded")
			}

			d.resetRows()

			for _, item := range cachedFeed.GetItemsOrderedByDate() {
				d.appendToRaw(item.Url)
			}

			d.currentSection = ARTICLES_LIST
			d.currentFeedUrl = url

			d.renderArticleList()

			var browserHelp string
			if !util.IsHeadless() {
				browserHelp = " | o = open with browser"
			}

			var lynxHelp string
			if util.IsLynxPresent() {
				lynxHelp = " | l = open with lynx"
			}

			d.setTopMessage(fmt.Sprintf("> %s", cachedFeed.Name))
			d.setBottomMessage(fmt.Sprintf("%s %s %s", articlesListSectionMsg, browserHelp, lynxHelp))

			go func() {
				if err := d.cache.Encode(); err != nil {
					log.Default().Println(err.Error())
				}
			}()
		}
	}
	if !found {
		d.setTmpBottomMessage(3*time.Second, "feed not yet loaded: press r!")
		return fmt.Errorf("cannot find articles of feed with url: %s", url)
	}
	return nil
}

func (d *display) loadArticleText(url string) error {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {

		if cachedFeed.Url == d.currentFeedUrl {

			for _, i := range cachedFeed.Items {

				if i.Url == url {

					text, err := html.ExtractText(i.Url)
					if err != nil {
						log.Default().Println(err)
						d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load article from url: %s", url))
						return fmt.Errorf("cannot load aricle")
					}

					i.Unread = false

					d.resetRows()

					scanner := bufio.NewScanner(strings.NewReader(text))
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if line != "" {
							d.raw = append(d.raw, []byte(line+"\n"))
						}
					}

					d.renderArticleText()

					d.currentArticleUrl = url
					d.currentSection = ARTICLE_TEXT

					d.setTopMessage(fmt.Sprintf("> %s > %s", cachedFeed.Name, i.Title))
					d.setBottomMessage(articleTextSectionMsg)

					go func() {
						if err := d.cache.Encode(); err != nil {
							log.Default().Println(err.Error())
						}
					}()

					break
				}
			}
		}
	}
	return nil
}

func (d *display) addNewFeed() {

	url := strings.TrimSpace(strings.Join(d.editingBuf, ""))

	for _, f := range d.config.Feeds {
		if f.Url == url {
			d.setTmpBottomMessage(3*time.Second, "already added!")
			return
		}
	}

	parsedFeed, err := d.fetchFeed(url)
	if err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, "cannot parse feed!")
		return
	}

	if err := d.config.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, "cannot add new feed to config!")
		return
	}

	if err := d.config.Encode(); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, "cannot write new feed to config!")
		return
	}

	d.appendToRaw(url)

	d.current.cx = 1
	d.current.cy = len(d.raw) % d.getContentWindowLen()
	d.current.startoff = (len(d.raw) - 1) / d.getContentWindowLen() * d.getContentWindowLen()

	d.setBottomMessage(urlsListSectionMsg)
	d.setTmpBottomMessage(3*time.Second, "new feed saved!")
	d.exitEditingMode()
}
