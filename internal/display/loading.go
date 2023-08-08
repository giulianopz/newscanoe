package display

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/internal/html"
	"github.com/giulianopz/newscanoe/internal/util"
)

func (d *display) LoadFeedList() error {

	d.mu.Lock()
	defer d.mu.Unlock()

	d.resetRows()

	filePath, err := util.GetUrlsFilePath()
	if err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	empty, err := util.IsEmpty(file)
	if err != nil {
		return err
	}

	if empty && len(d.cache.GetFeeds()) == 0 {
		d.setBottomMessage("no feed url: type 'a' to add one now")
	} else {

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {

			url := scanner.Bytes()
			if !strings.Contains(string(url), "#") {
				d.appendToRaw(string(url))
			}
		}
		d.setBottomMessage(urlsListSectionMsg)
	}

	//d.resetCurrentPos()

	d.renderURLs()

	d.setTopMessage("")

	d.current.cy = 1
	d.current.cx = 1
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

	if err := d.cache.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
		return
	}

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

		if err := d.cache.AddFeed(parsedFeed, url); err != nil {
			log.Default().Println(err)
			d.setTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
			return
		}

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

func (d *display) loadArticleList(url string) error {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == url {

			if len(cachedFeed.Items) == 0 {
				d.setTmpBottomMessage(3*time.Second, "feed not yet loaded: press r!")
				return fmt.Errorf("feed not loaded")
			}

			cachedFeed.New = false

			d.resetRows()

			for _, item := range cachedFeed.GetItemsOrderedByDate() {
				d.appendToRaw(item.Url)
			}

			//d.resetCurrentPos()

			d.currentSection = ARTICLES_LIST
			d.currentFeedUrl = url

			d.renderArticlesList()

			var browserHelp string
			if !util.IsHeadless() {
				browserHelp = " | o = open with browser"
			}

			var lynxHelp string
			if util.IsLynxPresent() {
				lynxHelp = " | l = open with lynx"
			}

			d.setTopMessage(fmt.Sprintf("> %s", cachedFeed.Title))
			d.setBottomMessage(fmt.Sprintf("%s %s %s", articlesListSectionMsg, browserHelp, lynxHelp))

			go func() {
				if err := d.cache.Encode(); err != nil {
					log.Default().Println(err.Error())
				}
			}()
		}
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

					i.New = false

					d.resetRows()

					scanner := bufio.NewScanner(strings.NewReader(text))
					for scanner.Scan() {
						line := strings.TrimSpace(scanner.Text())
						if line != "" {
							d.raw = append(d.raw, []byte(line+"\n"))
						}
					}

					//d.resetCurrentPos()

					d.renderArticleText()

					d.currentArticleUrl = url
					d.currentSection = ARTICLE_TEXT

					d.setTopMessage(fmt.Sprintf("> %s > %s", cachedFeed.Title, i.Title))
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

func (d *display) addEnteredFeedUrl() {

	url := strings.TrimSpace(strings.Join(d.editingBuf, ""))

	if !d.canBeParsed(url) {
		d.setTmpBottomMessage(3*time.Second, "feed url not valid!")
		return
	}

	if err := util.AppendUrl(url); err != nil {
		log.Default().Println(err)

		var target *util.UrlAlreadyPresentErr
		if errors.As(err, &target) {
			d.setTmpBottomMessage(3*time.Second, err.Error())
			return
		}
		d.setTmpBottomMessage(3*time.Second, "cannot save url in config file!")
		return
	}

	d.appendToRaw(url)

	d.current.cx = 1
	d.current.cy = len(d.raw) % d.getContentWindowLen()
	d.current.startoff = (len(d.raw) - 1) / d.getContentWindowLen() * d.getContentWindowLen()

	d.loadFeed(url)

	d.setBottomMessage(urlsListSectionMsg)
	d.setTmpBottomMessage(3*time.Second, "new feed saved!")
	d.exitEditingMode()
}
