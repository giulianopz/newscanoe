package display

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/termios"
	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sys/unix"
)

var DebugMode bool

const (
	// keys
	ENTER      = 13
	BACKSPACE  = 127
	ARROW_LEFT = iota
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DEL_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
	QUIT
	// sections
	URLS_LIST = iota
	ARTICLES_LIST
	ARTICLE_TEXT
	//
	bottomPadding int = 3
)

type Display struct {
	// cursor's position within terminal window
	cx int
	cy int
	// offsets of rendered text window
	startoff int
	endoff   int
	// size of terminal window
	height int
	width  int

	statusMsg string
	infotime  time.Time

	origTermios unix.Termios

	rows     [][]byte
	rendered [][]byte
	cache    cache.Cache

	/* 	totalsRows     int
	   	currentRow     int */
	currentSection    int
	currentArticleUrl string
	currentFeedUrl    string
}

func New(in uintptr) *Display {
	d := &Display{
		cx:       1,
		cy:       1,
		startoff: 0,
		endoff:   0,
		cache:    cache.Cache{},
	}

	d.SetWindowSize(in)

	if err := d.LoadCache(); err != nil {
		panic(err)
	}

	if err := d.LoadURLs(); err != nil {
		panic(err)
	}

	return d
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

func (d *Display) MoveCursor(dir byte) {
	switch dir {
	case ARROW_LEFT:
		if d.cx > 1 {
			d.cx--
		} else if d.cy > 1 {
			d.cy--
			d.cx = len(d.rendered[d.cy-1+d.startoff])
		}
	case ARROW_RIGHT:
		if (d.cx - 1) < (len(d.rendered[d.cy-1+d.startoff]) - 1) {
			d.cx++
		} else if d.cy >= 1 && d.cy < (d.height-bottomPadding) {
			d.cy++
			d.cx = 1
		}
	case ARROW_DOWN:
		if d.cy < (d.height - bottomPadding) {
			if (d.cx - 1) <= (len(d.rendered[d.cy+d.startoff]) - 1) {
				d.cy++
			}
		} else if d.endoff < len(d.rendered)-1 {
			d.startoff++
		}
	case ARROW_UP:
		if d.cy > 1 {
			if (d.cx - 1) <= (len(d.rendered[d.cy-2+d.startoff]) - 1) {
				d.cy--
			}
		} else if d.startoff > 0 {
			d.startoff--
		}
	}
}

func readKeyStroke(fd uintptr) byte {

	input := make([]byte, 1)
	for {

		_, err := unix.Read(int(fd), input)
		if err != nil {
			panic(err)
		}

		if input[0] == '\x1b' {

			seq := make([]byte, 3)

			_, err := unix.Read(int(fd), seq)
			if err != nil {
				return QUIT
			}

			if seq[0] == '[' {

				if seq[1] >= '0' && seq[1] <= '9' {
					_, err = unix.Read(int(fd), seq[2:])
					if err != nil {
						return QUIT
					}

					if seq[2] == '~' {
						switch seq[1] {
						case '1':
							return HOME_KEY
						case '3':
							return DEL_KEY
						case '4':
							return END_KEY
						case '5':
							return PAGE_UP
						case '6':
							return PAGE_DOWN
						case '7':
							return HOME_KEY
						case '8':
							return END_KEY
						}
					}
				} else {

					switch seq[1] {
					case 'A':
						return ARROW_UP
					case 'B':
						return ARROW_DOWN
					case 'C':
						return ARROW_RIGHT
					case 'D':
						return ARROW_LEFT
					case 'H':
						return HOME_KEY
					case 'F':
						return END_KEY
					}
				}
			} else if seq[0] == 'O' {

				switch seq[1] {
				case 'H':
					return HOME_KEY
				case 'F':
					return END_KEY
				}
			}
			return QUIT
		} else {
			return input[0]
		}
	}
}

func (d *Display) Quit(quitC chan bool) {
	fmt.Fprint(os.Stdout, "\x1b[2J")
	fmt.Fprint(os.Stdout, "\x1b[H")
	quitC <- true
}

func (d *Display) ProcessKeyStroke(fd uintptr, quitC chan bool) {

	input := readKeyStroke(fd)

	switch input {

	case ctrlPlus('q'), 'q':
		d.Quit(quitC)

	case ctrlPlus('r'), 'r':
		if d.currentSection == URLS_LIST {
			d.LoadFeed(string(d.rows[d.cy-1+d.startoff]))
			d.RefreshScreen()
		}

	case ctrlPlus('o'), 'o':
		if d.currentSection == ARTICLE_TEXT {

			if !headless() {
				d.openWithBrowser(d.currentArticleUrl)
			}
		}

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.MoveCursor(input)

	case ENTER:
		{
			switch d.currentSection {
			case URLS_LIST:
				d.LoadArticlesList(string(d.rows[d.cy-1+d.startoff]))
			case ARTICLES_LIST:
				d.LoadArticle(string(d.rows[d.cy-1+d.startoff]))
			}
		}

	case BACKSPACE:
		{
			switch d.currentSection {
			case ARTICLES_LIST:
				d.LoadURLs()
				d.currentFeedUrl = ""
			case ARTICLE_TEXT:
				d.LoadArticlesList(d.currentFeedUrl)
				d.currentArticleUrl = ""
			}
		}

		/* 	default:
		d.SetStatusMessage(fmt.Sprintf("keystroke: %v", input)) */
	}
}

func (d *Display) RefreshScreen() {

	buf := &bytes.Buffer{}

	// erase entire screen
	buf.WriteString("\x1b[2J")

	// hide cursor
	buf.WriteString("\x1b[?25l")
	buf.WriteString("\x1b[H")

	d.Draw(buf)

	// move cursor to (y,x)
	buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", d.cy, d.cx))
	// show cursor
	buf.WriteString("\x1b[?25h")

	fmt.Fprint(os.Stdout, buf)
}

func (d *Display) SetStatusMessage(msg string) {
	d.statusMsg = msg
}

func (d *Display) SetTmpStatusMessage(t time.Duration, msg string) {
	previous := d.statusMsg
	d.SetStatusMessage(msg)
	go func() {
		time.AfterFunc(t, func() {
			d.SetStatusMessage(previous)
		})
	}()
}

func (d *Display) SetWindowSize(fd uintptr) error {
	w, h, err := termios.GetWindowSize(int(fd))
	if err != nil {
		return err
	}
	d.width = w
	d.height = h
	return nil
}

func (d *Display) LoadCache() error {
	cachePath, err := util.GetCacheFilePath()
	if err != nil {
		return err
	}

	if util.Exists(cachePath) {
		if err := d.cache.Decode(); err != nil {
			return err
		}
	}
	return nil
}

func (d *Display) LoadURLs() error {

	d.rows = make([][]byte, 0)
	d.rendered = make([][]byte, 0)

	filePath, err := util.GetUrlsFilePath()
	if err != nil {
		// TODO
		panic(err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	cached := make(map[string]bool, 0)

	if len(d.cache.Feeds) == 0 {
		d.cache.Feeds = make([]*cache.Feed, 0)
	} else {
		for _, cachedFeed := range d.cache.Feeds {
			cached[cachedFeed.Url] = true
			d.rows = append(d.rows, []byte(cachedFeed.Url))
			d.rendered = append(d.rendered, []byte(cachedFeed.Title))
		}
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		url := scanner.Bytes()
		if !strings.Contains(string(url), "#") {

			if _, present := cached[string(url)]; !present {
				d.rows = append(d.rows, url)
				d.rendered = append(d.rendered, url)
				d.cache.Feeds = append(d.cache.Feeds,
					&cache.Feed{
						Url:   string(url),
						Title: string(url),
					})
			}
		}
	}

	d.cy = 1
	d.cx = 1
	d.currentSection = URLS_LIST
	// Ctrl-R/R = reload all
	d.SetStatusMessage("HELP: Ctrl-q/q = quit | Ctrl-r/r = reload")

	return nil
}

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 \.\-]+`)

func (d *Display) LoadFeed(url string) {

	fp := gofeed.NewParser()
	parsedFeed, err := fp.ParseURL(url)
	if err != nil {
		// TODO
		panic(err)
	}

	title := nonAlphaNumericRegex.ReplaceAllString(parsedFeed.Title, "")
	title = strings.Trim(title, " ")

	var cached int
	for _, cachedFeed := range d.cache.Feeds {
		if cachedFeed.Url == url {
			cachedFeed.Title = title
			cachedFeed.Items = make([]*cache.Item, 0)

			for _, parsedItem := range parsedFeed.Items {
				cachedItem := &cache.Item{
					Title:   parsedItem.Title,
					Url:     parsedItem.Link,
					PubDate: *parsedItem.PublishedParsed,
				}
				cachedFeed.Items = append(cachedFeed.Items, cachedItem)
				cached++
			}

			d.rendered[d.cy-1+d.startoff] = []byte(title)
			d.currentFeedUrl = url
		}
	}

	if cached != 0 {
		if err := d.cache.Encode(); err != nil {
			d.SetStatusMessage(err.Error())
		}
	}
}

func (d *Display) LoadArticlesList(url string) {

	for _, cachedFeed := range d.cache.Feeds {
		if cachedFeed.Url == url {

			if len(cachedFeed.Items) == 0 {
				d.SetTmpStatusMessage(3*time.Second, "feed not yet loaded: press r!")
				return
			}

			d.rows = make([][]byte, 0)
			d.rendered = make([][]byte, 0)

			for _, item := range cachedFeed.Items {

				d.rows = append(d.rows, []byte(item.Url))
				dateAndArticleName := fmt.Sprintf("%-20s %s", item.PubDate.Format("2006-January-02"), item.Title)
				d.rendered = append(d.rendered, []byte(dateAndArticleName))
			}
			d.cy = 1
			d.cx = 1
			d.currentSection = ARTICLES_LIST
			d.currentFeedUrl = url
			d.SetStatusMessage("HELP: Enter = view article | Backspace = go back")
		}
	}
}

func (d *Display) LoadArticle(url string) {

	for _, cachedFeed := range d.cache.Feeds {
		if cachedFeed.Url == d.currentFeedUrl {

			for _, i := range cachedFeed.Items {

				if i.Url == url {

					resp, err := http.Get(i.Url)
					if err != nil {
						panic(err)
					}
					defer resp.Body.Close()

					converter := md.NewConverter("", true, nil)
					markdown, err := converter.ConvertReader(resp.Body)
					if err != nil {
						panic(err)
					}

					d.rows = make([][]byte, 0)
					d.rendered = make([][]byte, 0)

					for _, line := range strings.Split(markdown.String(), "\n") {

						line := strings.Trim(line, " ")
						if line != "" {

							if len(line) > d.width {

								tmp := make([]byte, 0)
								for _, r := range line {
									if len(tmp) >= d.width {
										d.rows = append(d.rows, tmp)
										d.rendered = append(d.rendered, tmp)
										tmp = make([]byte, 0)
									}
									tmp = append(tmp, byte(r))
								}
								if len(tmp) != 0 {
									d.rows = append(d.rows, tmp)
									d.rendered = append(d.rendered, tmp)
								}

							} else {
								d.rows = append(d.rows, []byte(line))
								d.rendered = append(d.rendered, []byte(line))
							}
						}
					}
				}

				d.cy = 1
				d.cx = 1
				d.currentArticleUrl = url
				d.currentSection = ARTICLE_TEXT

				var browserHelp string
				if !headless() {
					browserHelp = "Ctrl-o/o = open with browser |"
				}
				d.SetStatusMessage(fmt.Sprintf("HELP: %s Backspace = go back", browserHelp))
			}
		}
	}
}

func (d *Display) Draw(buf *bytes.Buffer) {

	d.endoff = (len(d.rendered) - 1)
	if d.endoff > (d.height - bottomPadding) {
		d.endoff = d.height - bottomPadding - 1
	}
	d.endoff += d.startoff

	var printed int
	for i := d.startoff; i <= d.endoff; i++ {

		for j := 0; j < len(d.rendered[i]); j++ {
			if j < (d.width) {
				buf.WriteString(string(d.rendered[i][j]))
			} else {
				// TODO
				d.SetStatusMessage("char is beyond win width")
			}
		}
		buf.WriteString("\r\n")
		printed++
	}

	for ; printed < d.height-bottomPadding; printed++ {
		buf.WriteString("\r\n")
	}

	for k := 0; k < d.width; k++ {
		buf.WriteString("-")
	}
	buf.WriteString("\r\n")

	var tracking string
	if DebugMode {
		tracking = fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.cy, d.cx, d.startoff, d.endoff, d.height, d.width)
	}
	buf.WriteString(fmt.Sprintf("%s\t%s\r\n", d.statusMsg, tracking))
}

/*
openWithBrowser opens the current viewed article with the default browser for desktop environment.
The environment variable DISPLAY is used to detect if the app is running on a headless machine.
see: https://wiki.debian.org/DefaultWebBrowser
*/
func (d *Display) openWithBrowser(url string) {

	if url != "" {
		err := exec.Command("xdg-open", url).Run()
		if err != nil {
			switch e := err.(type) {
			case *exec.Error:
				d.SetStatusMessage(fmt.Sprintf("failed executing: %v", err))
			case *exec.ExitError:
				d.SetStatusMessage(fmt.Sprintf("command exit with code: %v", e.ExitCode()))
			default:
				panic(err)
			}
		}
	}
}

// headless detects if this is a headless machine by looking up the DISPLAY environment variable
func headless() bool {
	displayVar, set := os.LookupEnv("DISPLAY")
	return !set || displayVar == ""
}
