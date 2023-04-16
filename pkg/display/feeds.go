package display

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/escape"
	"github.com/giulianopz/newscanoe/pkg/util"
)

func (d *display) LoadURLs() error {

	d.mu.Lock()
	defer d.mu.Unlock()

	d.resetRows()

	filePath, err := util.GetUrlsFilePath()
	if err != nil {
		log.Panicln(err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		log.Panicln(err)
	}
	defer file.Close()

	empty, err := util.IsEmpty(file)
	if err != nil {
		log.Panicln(err)
	}

	if empty && len(d.cache.GetFeeds()) == 0 {
		d.setBottomMessage("no feed url: type 'a' to add one now")
	} else {
		cached := make(map[string]*cache.Feed, 0)

		for _, cachedFeed := range d.cache.GetFeeds() {
			cached[cachedFeed.Url] = cachedFeed
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {

			url := scanner.Bytes()
			if !strings.Contains(string(url), "#") {

				cachedFeed, present := cached[string(url)]
				if !present {
					d.appendToRaw(string(url))
					d.appendToRendered(string(url))
				} else {
					d.appendToRaw(cachedFeed.Url)
					d.appendToRendered(cachedFeed.Title)
				}
			}
		}
		d.setBottomMessage(urlsListSectionMsg)
	}

	d.cy = 1
	d.cx = 1
	d.currentSection = URLS_LIST
	return nil
}

func (d *display) loadFeed(url string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	parsedFeed, err := d.parser.ParseURL(url)
	if err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, "cannot parse feed!")
		return
	}

	title := strings.TrimSpace(parsedFeed.Title)

	if err := d.cache.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
		return
	}

	d.rendered[d.currentRow()] = []byte(title)
	d.currentFeedUrl = url

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()
}

func (d *display) loadAllFeeds() {

	d.mu.Lock()
	defer d.mu.Unlock()

	origMsg := d.bottomBarMsg

	for id := range d.raw {

		url := string(d.raw[id])

		log.Default().Printf("loading feed #%d from url %s\n", id, url)

		parsedFeed, err := d.parser.ParseURL(url)
		if err != nil {
			log.Default().Println(err)
			d.setTmpBottomMessage(3*time.Second, "cannot parse feed!")
			return
		}

		title := strings.TrimSpace(parsedFeed.Title)

		if err := d.cache.AddFeed(parsedFeed, url); err != nil {
			log.Default().Println(err)
			d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
			return
		}

		d.rendered[id] = []byte(title)

		d.setBottomMessage(fmt.Sprintf("loading all feeds, please wait........%d/%d", id+1, len(d.raw)))
		d.RefreshScreen()
	}

	d.setBottomMessage(origMsg)

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()

}

func (d *display) loadArticlesList(url string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == url {

			if len(cachedFeed.Items) == 0 {
				d.setTmpBottomMessage(3*time.Second, "feed not yet loaded: press r!")
				return
			}

			d.resetRows()

			for _, item := range cachedFeed.GetItemsOrderedByDate() {
				d.appendToRaw(item.Url)
				d.appendToRendered(util.RenderArticleRow(item.PubDate, item.Title))
			}

			d.resetCoordinates()

			d.currentSection = ARTICLES_LIST
			d.currentFeedUrl = url

			var browserHelp string
			if !util.IsHeadless() {
				browserHelp = " | o = open with browser"
			}

			var lynxHelp string
			if util.IsLynxPresent() {
				lynxHelp = " | l = open with lynx"
			}

			d.setBottomMessage(fmt.Sprintf("%s %s %s", articlesListSectionMsg, browserHelp, lynxHelp))
		}
	}
}

func (d *display) loadArticleText(url string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == d.currentFeedUrl {

			for _, i := range cachedFeed.Items {

				if i.Url == url {

					resp, err := d.client.Get(i.Url)
					if err != nil {
						log.Default().Println(err)
						d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load article from url: %s", url))
						return
					}
					defer resp.Body.Close()

					converter := md.NewConverter("", true, nil)
					markdown, err := converter.ConvertReader(resp.Body)
					if err != nil {
						log.Default().Println(err)
						d.setTmpBottomMessage(3*time.Second, "cannot parse article text!")
						return
					}

					d.resetRows()

					scanner := bufio.NewScanner(&markdown)
					for scanner.Scan() {
						d.raw = append(d.raw, []byte(scanner.Text()))
					}

					d.renderText()

					d.resetCoordinates()

					d.currentArticleUrl = url
					d.currentSection = ARTICLE_TEXT

					d.setBottomMessage(articleTextSectionMsg)
					break
				}
			}
		}
	}
}

func (d *display) addEnteredFeedUrl() {

	url := strings.TrimSpace(strings.Join(d.editingBuf, ""))

	if !d.canBeParsed(url) {
		d.bottomBarColor = escape.RED
		d.setTmpBottomMessage(3*time.Second, "feed url not valid!")
		return
	}

	if err := util.AppendUrl(url); err != nil {
		log.Default().Println(err)

		d.bottomBarColor = escape.RED

		var target *util.UrlAlreadyPresentErr
		if errors.As(err, &target) {
			d.setTmpBottomMessage(3*time.Second, err.Error())
			return
		}
		d.setTmpBottomMessage(3*time.Second, "cannot save url in config file!")
		return
	}

	d.appendToRaw(url)
	d.appendToRendered(url)

	d.cx = 1
	d.cy = len(d.rendered) % (d.height - BOTTOM_PADDING)
	d.startoff = (len(d.rendered) - 1) / (d.height - BOTTOM_PADDING) * (d.height - BOTTOM_PADDING)

	d.loadFeed(url)

	d.setBottomMessage(urlsListSectionMsg)
	d.setTmpBottomMessage(3*time.Second, "new feed saved!")
	d.exitEditingMode(escape.GREEN)
}