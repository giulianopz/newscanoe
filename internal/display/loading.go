package display

import (
	"fmt"
	"log"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/internal/bar"
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

		// TODO sort only once
		sort.SliceStable(d.config.Feeds, func(i, j int) bool {
			return strings.ToLower(d.config.Feeds[i].Name) < strings.ToLower(d.config.Feeds[j].Name)
		})
		for _, f := range d.config.Feeds {

			//TODO use only the cache obj when the app is started
			// until then, the unread count will be wrong

			d.appendRow(feedRow(f.UnreadCount, len(f.Items), f.Name))
		}
		d.setBottomMessage(urlsListSectionMsg)
	}

	//TODO d.renderFeedList()

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
		d.setTmpBottomMessage(2*time.Second, "cannot parse feed!")
		return nil, err
	}

	parsedFeed = d.cache.AddFeed(parsedFeed, url)

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

	pb := bar.NewProgressBar(d.height, 1, d.width, len(d.rows))

	g := new(errgroup.Group)
	g.SetLimit(-1)

	// TODO get all urls from somewhere else
	urls := d.rows
	for len(urls) > 0 {

		//TODO
		//url := urls[0]

		g.Go(func() error {

			//TODO
			log.Default().Printf("loading feed url: %s\n", "")

			//TODO
			parsedFeed, err := d.parser.Parse("")
			if err != nil {
				log.Default().Println(err)
				return err
			}
			//TODO
			d.cache.AddFeed(parsedFeed, "")

			pb.IncrByOne()

			return nil
		})

		urls = urls[1:]
	}

	if err := g.Wait(); err != nil {
		slog.Error("reload failed", "err", err)
		d.setTmpBottomMessage(2*time.Second, "cannot reload all feeds!")
		return
	}

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()

	log.Default().Println("reloaded all feeds in: ", time.Since(start))
}

func clean(s string) string {
	bs := []byte(s)
	n := 0
	for _, r := range bs {
		if ('a' <= r && r <= 'z') ||
			('A' <= r && r <= 'Z') ||
			('0' <= r && r <= '9') {
			bs[n] = r
			n++
		}
	}
	return string(bs[:n])
}

func (d *display) loadArticleList(i int) error {

	d.mu.Lock()
	defer d.mu.Unlock()

	f := d.config.Feeds[i]

	var cachedFeed *feed.Feed
	for _, c := range d.cache.GetFeeds() {
		if c.Url == f.Url {
			cachedFeed = c
		}
	}

	if cachedFeed == nil {
		return fmt.Errorf("cannot find cached feed")
	}

	if len(cachedFeed.Items) == 0 {
		d.setTmpBottomMessage(2*time.Second, "feed not yet loaded: press r!")
		return fmt.Errorf("feed not loaded")
	}

	d.resetRows()

	for _, item := range cachedFeed.GetItemsOrderedByDate() {
		//TODO if unread, apply bold
		d.appendRow(articleRow(item.PubDate, item.Title))
	}

	d.currentSection = ARTICLES_LIST
	d.currentFeedUrl = cachedFeed.Url

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

	return nil
}

func (d *display) loadArticleText() error {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {

		if cachedFeed.Url == d.currentFeedUrl {

			i := cachedFeed.GetItemsOrderedByDate()[d.currentArrIdx()]

			// extract text with go-html2ansi as readability serializer
			text, err := html.ExtractText(i.Url)
			if err != nil {
				log.Default().Println(err)
				d.setTmpBottomMessage(2*time.Second, "cannot load article")
				return fmt.Errorf("cannot load aricle")
			}

			i.Unread = false

			d.resetRows()

			d.reflow(text)

			d.currentSection = ARTICLE_TEXT

			d.setTopMessage(fmt.Sprintf("> %s > %s", cachedFeed.Name, i.Title))
			d.setBottomMessage(articleTextSectionMsg)

			go func() {
				if err := d.cache.Encode(); err != nil {
					log.Default().Println(err.Error())
				}
			}()
		}
	}
	return nil
}

func (d *display) addNewFeed() {

	url := strings.TrimSpace(d.editingBuf.String())

	for _, f := range d.config.Feeds {
		if f.Url == url {
			d.setTmpBottomMessage(2*time.Second, "already added!")
			return
		}
	}

	parsedFeed, err := d.fetchFeed(url)
	if err != nil {
		log.Default().Println(err)
		return
	}

	if err := d.config.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(2*time.Second, "cannot add new feed to config!")
		return
	}

	if err := d.config.Encode(); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(2*time.Second, "cannot write new feed to config!")
		return
	}

	d.cache.Merge(d.config)

	d.appendRow(url)

	d.resetRows()

	sort.SliceStable(d.config.Feeds, func(i, j int) bool {
		return strings.ToLower(d.config.Feeds[i].Name) < strings.ToLower(d.config.Feeds[j].Name)
	})
	for _, f := range d.config.Feeds {
		d.appendRow(f.Url)
	}

	//TODO d.renderFeedList()

	idx := d.indexOf(url)
	if idx == -1 {
		log.Default().Println("cannot find url:", url)
		d.setTmpBottomMessage(2*time.Second, "cannot find the find you added!")
		return
	}
	idx++

	d.current.cx = 1
	max := d.getContentWindowLen()
	d.current.cy = idx % max
	d.current.startoff = idx / max * max

	d.setBottomMessage(urlsListSectionMsg)
	d.setTmpBottomMessage(2*time.Second, "new feed saved!")
	d.exitEditingMode()
}
