package display

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	md "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/giulianopz/newscanoe/pkg/ascii"
	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/escape"
	"github.com/giulianopz/newscanoe/pkg/termios"
	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sys/unix"
)

var DebugMode bool

const (
	/* keys */

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

	/* display sections */

	URLS_LIST = iota
	ARTICLES_LIST
	ARTICLE_TEXT

	// num of lines reserved to bottom bar plus final empty row
	BOTTOM_PADDING = 3
	// colors
	WHITE = 37
	RED   = 31
	GREEN = 32
)

const (
	urlsListSectionMsg     = "HELP: q = quit | r = reload | R = reload all | a = add a feed"
	articlesListSectionMsg = "HELP: Enter = view article | Backspace = go back"
	articleTextSectionMsg  = "HELP: Backspace = go back"
)

type display struct {
	// cursor's position within terminal window
	cx int
	cy int
	// offsets of rendered text window
	startoff int
	endoff   int
	// size of terminal window
	height int
	width  int

	bottomBarColor int
	// message displayed in the bottom bar
	bottomBarMsg string
	// message displayed in the right corner of the bottom bar
	bottomRightCorner string

	// previous state of the terminal
	origTermios unix.Termios

	mu sync.Mutex
	// dislay raw text
	raw [][]byte
	// dislay rendered text
	rendered [][]byte
	// gob cache
	cache *cache.Cache

	editingMode bool
	editingBuf  []string

	currentSection    int
	currentArticleUrl string
	currentFeedUrl    string

	Quitting bool
}

func New(in uintptr) *display {
	d := &display{
		cx:             1,
		cy:             1,
		startoff:       0,
		endoff:         0,
		cache:          cache.NewCache(),
		bottomBarColor: escape.WHITE,
	}

	d.SetWindowSize(in)

	if err := d.loadCache(); err != nil {
		log.Panicln(err)
	}

	if err := d.loadURLs(); err != nil {
		log.Panicln(err)
	}

	return d
}

func (d *display) Quit(quitC chan bool) {

	log.Default().Println("quitting")
	d.Quitting = true

	fmt.Fprint(os.Stdout, escape.SHOW_CURSOR)
	fmt.Fprint(os.Stdout, escape.ERASE_ENTIRE_SCREEN)
	fmt.Fprint(os.Stdout, escape.MoveCursor(1, 1))
	quitC <- true
}

func (d *display) appendRow(raw, rendered string) {
	d.raw = append(d.raw, []byte(raw))
	d.rendered = append(d.rendered, []byte(rendered))
}

func (d *display) currentRow() int {
	return d.cy - 1 + d.startoff
}

func (d *display) resetCoordinates() {
	d.cy = 1
	d.cx = 1
	d.startoff = 0
}

func (d *display) resetRows() {
	d.raw = make([][]byte, 0)
	d.rendered = make([][]byte, 0)
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

func (d *display) moveCursor(dir byte) {
	switch dir {
	/* case ARROW_LEFT:
		if d.cx > 1 {
			d.cx--
		} else if d.cy > 1 {
			d.cy--
			d.cx = len(d.rendered[d.currentRow()])
		}
	case ARROW_RIGHT:
		if (d.cx - 1) < (len(d.rendered[d.currentRow()]) - 1) {
			d.cx++
		} else if d.cy >= 1 && d.cy < (d.height-BOTTOM_PADDING) && d.cy+1 < len(d.rendered) {
			d.cy++
			d.cx = 1
		} */
	case ARROW_DOWN:
		if d.cy < (d.height - BOTTOM_PADDING) {
			if d.currentRow()+1 <= len(d.rendered)-1 && (d.cx-1) <= (len(d.rendered[d.currentRow()+1])-1) {
				d.cy++
			}
		} else if d.endoff < len(d.rendered)-1 {
			d.startoff++
		}
	case ARROW_UP:
		if d.cy > 1 {
			if d.currentRow()-1 <= len(d.rendered)-1 && (d.cx-1) <= (len(d.rendered[d.currentRow()-1])-1) {
				d.cy--
			}
		} else if d.startoff > 0 {
			d.startoff--
		}
	}
}

func (d *display) scroll(dir byte) {
	switch dir {
	case PAGE_DOWN:
		{
			if d.endoff == len(d.rendered)-1 {
				d.cy = d.height - BOTTOM_PADDING
				return
			}

			firstItemInNextPage := d.endoff + 1
			if firstItemInNextPage < len(d.rendered)-1 {
				d.startoff = firstItemInNextPage
			} else {
				d.startoff++
				d.endoff = len(d.rendered) - 1
			}

			d.cy = d.height - BOTTOM_PADDING
		}
	case PAGE_UP:
		{
			if d.startoff == 0 {
				d.cy = 1
				return
			}

			firstItemInPreviousPage := d.startoff - (d.height - BOTTOM_PADDING)
			if firstItemInPreviousPage >= 0 {
				d.startoff = firstItemInPreviousPage
			} else {
				d.startoff = 0
				d.endoff = d.height - BOTTOM_PADDING - 1
			}

			d.cy = 1
		}
	}
}

func readKeyStroke(fd uintptr) byte {

	input := make([]byte, 1)
	for {

		_, err := unix.Read(int(fd), input)
		if err != nil {
			//TODO use new slog
			log.Default().Println(err)
		}
		log.Default().Printf("1st keystroke byte: %v", input)

		if input[0] == ascii.ESC {

			seq := make([]byte, 3)

			_, err := unix.Read(int(fd), seq)
			if err != nil {
				return QUIT
			}

			log.Default().Printf("2nd keystroke byte: %v", seq[0])

			// 91
			if seq[0] == '[' {

				log.Default().Printf("3rd keystroke byte: %v", seq[1])
				log.Default().Printf("4th keystroke byte: %v", seq[2])

				// 48 and 57
				if seq[1] >= '0' && seq[1] <= '9' {
					// 126
					if seq[2] == '~' {
						switch seq[1] {
						case '1': // 49
							return HOME_KEY
						case '3': // 51
							return DEL_KEY
						case '4': // 52
							return END_KEY
						case '5': // 53
							return PAGE_UP
						case '6': // 54
							return PAGE_DOWN
						case '7': // 55
							return HOME_KEY
						case '8': // 56
							return END_KEY
						}
					}
				} else {

					switch seq[1] {
					case 'A': // 65
						return ARROW_UP
					case 'B': // 65
						return ARROW_DOWN
					case 'C': // 67
						return ARROW_RIGHT
					case 'D': // 68
						return ARROW_LEFT
					case 'H': // 72
						return HOME_KEY
					case 'F': // 70
						return END_KEY
					}
				}
			} else if seq[0] == 'O' { // 79

				switch seq[1] {
				case 'H': // 72
					return HOME_KEY
				case 'F': // 70
					return END_KEY
				}
			}
			return QUIT
		} else {
			return input[0]
		}
	}
}

func (d *display) ProcessKeyStroke(fd uintptr, quitC chan bool) {

	input := readKeyStroke(fd)

	// TODO copy from clipboard

	if d.editingMode {
		switch {

		case input == ARROW_LEFT:
			if d.cx > 1 {
				d.cx--
			}

		case input == ARROW_RIGHT:
			if d.cx < len(d.editingBuf)+1 {
				d.cx++
			}

		case input == DEL_KEY:
			{
				i := d.cx - 1
				if i < len(d.editingBuf) {
					d.deleteCharAt(i)
					d.setBottomMessage(strings.Join(d.editingBuf, ""))
				}
			}

		case input == ascii.BACKSPACE:
			{
				if len(d.editingBuf) != 0 && d.cx == len(d.editingBuf)+1 {
					d.deleteCharAt(len(d.editingBuf) - 1)
					d.setBottomMessage(strings.Join(d.editingBuf, ""))
					d.cx--
				}
			}
		case input == ascii.ENTER:
			{
				d.addEnteredFeedUrl()
			}
		case util.IsLetter(input), util.IsDigit(input), util.IsSpecialChar(input):
			{
				if d.cx < (d.width - utf8.RuneCountInString(d.bottomRightCorner) - 1) {
					i := d.cx - 1
					d.insertCharAt(string(input), i)
					d.setBottomMessage(strings.Join(d.editingBuf, ""))
					d.cx++
				}
			}
		case input == QUIT:
			{
				d.setBottomMessage(urlsListSectionMsg)
				d.setTmpBottomMessage(1*time.Second, "editing aborted!")
				d.exitEditingMode(escape.WHITE)
				d.resetCoordinates()
			}
		default:
			{
				log.Default().Printf("unhandled: %v\n", input)
			}
		}
		return
	}

	switch input {

	case ctrlPlus('q'), 'q':
		d.Quit(quitC)

	case ctrlPlus('r'), 'r':
		if d.currentSection == URLS_LIST {
			d.loadFeed(string(d.raw[d.currentRow()]))
		}

	case ctrlPlus('a'), 'a':
		if d.currentSection == URLS_LIST {
			d.enterEditingMode()
		}

	case ctrlPlus('R'), 'R':
		if d.currentSection == URLS_LIST {
			d.loadAllFeeds()
		}

	case ctrlPlus('o'), 'o':
		if d.currentSection == ARTICLES_LIST {
			if !util.IsHeadless() {
				if err := util.OpenWithBrowser(string(d.raw[d.currentRow()])); err != nil {
					d.setTmpBottomMessage(1*time.Second, err.Error())
				}
			}
		}

	case ctrlPlus('l'), 'l':
		if d.currentSection == ARTICLES_LIST {
			if util.IsLynxPresent() {
				if err := util.OpenWithLynx(string(d.raw[d.currentRow()])); err != nil {
					d.setTmpBottomMessage(1*time.Second, err.Error())
				}
			}
		}

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.moveCursor(input)

	case PAGE_UP, PAGE_DOWN:
		d.scroll(input)

	case ascii.ENTER:
		{
			url := string(d.raw[d.currentRow()])

			switch d.currentSection {
			case URLS_LIST:
				d.loadArticlesList(url)
			case ARTICLES_LIST:
				d.loadArticleText(url)
			}
		}

	case ascii.BACKSPACE:
		{
			switch d.currentSection {
			case ARTICLES_LIST:
				d.loadURLs()
				d.currentFeedUrl = ""
			case ARTICLE_TEXT:
				d.loadArticlesList(d.currentFeedUrl)
				d.currentArticleUrl = ""
			}
		}

	default:
		{
			log.Default().Printf("unhandled: %v\n", input)
		}
	}
}

func (d *display) addEnteredFeedUrl() {

	d.mu.Lock()
	defer d.mu.Unlock()

	url := strings.TrimSpace(strings.Join(d.editingBuf, ""))
	if err := d.isValidFeedUrl(url); err != nil {
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

	d.appendRow(url, url)

	d.cx = 1
	d.cy = len(d.rendered) % (d.height - BOTTOM_PADDING)
	d.startoff = (len(d.rendered) - 1) / (d.height - BOTTOM_PADDING) * (d.height - BOTTOM_PADDING)

	d.loadFeed(url)

	d.setBottomMessage(urlsListSectionMsg)
	d.setTmpBottomMessage(3*time.Second, "new feed saved!")
	d.exitEditingMode(escape.GREEN)
}

func (d *display) insertCharAt(c string, i int) {
	if i == len(d.editingBuf) {
		d.editingBuf = append(d.editingBuf, c)
	} else {
		d.editingBuf = append(d.editingBuf[:i+1], d.editingBuf[i:]...)
		d.editingBuf[i] = c
	}
}

func (d *display) deleteCharAt(i int) {
	if i == len(d.editingBuf)-1 {
		d.editingBuf[len(d.editingBuf)-1] = ""
		d.editingBuf = d.editingBuf[:len(d.editingBuf)-1]
	} else {
		copy(d.editingBuf[i:], d.editingBuf[i+1:])
		d.editingBuf[len(d.editingBuf)-1] = ""
		d.editingBuf = d.editingBuf[:len(d.editingBuf)-1]
	}
}

func (d *display) RefreshScreen() {

	log.Default().Println("refreshing screen")

	buf := &bytes.Buffer{}

	buf.WriteString(escape.ERASE_ENTIRE_SCREEN)
	buf.WriteString(escape.HIDE_CURSOR)
	buf.WriteString(escape.MoveCursor(1, 1))

	d.draw(buf)

	buf.WriteString(escape.MoveCursor(d.cy, d.cx))
	if d.editingMode {
		buf.WriteString(escape.SHOW_CURSOR)
	}

	fmt.Fprint(os.Stdout, buf)
}

func (d *display) setBottomMessage(msg string) {
	d.bottomBarMsg = msg
}

func (d *display) setTmpBottomMessage(t time.Duration, msg string) {
	previous := d.bottomBarMsg
	d.setBottomMessage(msg)
	go func() {
		time.AfterFunc(t, func() {
			d.bottomBarColor = escape.WHITE
			d.setBottomMessage(previous)
		})
	}()
}

func (d *display) SetWindowSize(fd uintptr) error {
	w, h, err := termios.GetWindowSize(int(fd))
	if err != nil {
		return err
	}
	d.width = w
	d.height = h
	return nil
}

func (d *display) loadCache() error {
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

func (d *display) exitEditingMode(color int) {
	d.editingMode = false
	d.editingBuf = []string{}
	d.bottomBarColor = color
}

func (d *display) enterEditingMode() {
	log.Default().Println("live editing enabled")

	d.editingMode = true
	d.editingBuf = []string{}

	d.cy = d.height - 1
	d.cx = 1

	d.setBottomMessage("")
}

func (d *display) loadURLs() error {

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
					d.appendRow(string(url), string(url))
				} else {
					d.appendRow(cachedFeed.Url, cachedFeed.Title)
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

func (d *display) isValidFeedUrl(url string) error {
	if _, err := gofeed.NewParser().ParseURL(url); err != nil {
		return err
	}
	return nil
}

func (d *display) loadFeed(url string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	parsedFeed, err := gofeed.NewParser().ParseURL(url)
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

		fp := gofeed.NewParser()
		parsedFeed, err := fp.ParseURL(url)
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
				d.appendRow(item.Url, util.RenderArticleRow(item.PubDate, item.Title))
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

var client = http.Client{
	Timeout: 3 * time.Second,
}

func (d *display) loadArticleText(url string) {

	d.mu.Lock()
	defer d.mu.Unlock()

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == d.currentFeedUrl {

			for _, i := range cachedFeed.Items {

				if i.Url == url {

					resp, err := client.Get(i.Url)
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

						line := strings.TrimSpace(scanner.Text())
						if line != "" {

							log.Default().Println("line: ", line)

							if len(line) > d.width {

								for i := 0; i < len(line); i += d.width {

									end := i + d.width
									if end > len(line) {
										end = len(line)
									}
									d.appendRow(line[i:end], line[i:end])
								}
							} else {
								d.appendRow(line, line)
							}
						}
					}

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

func (d *display) draw(buf *bytes.Buffer) {

	nextEndOff := d.startoff + (d.height - BOTTOM_PADDING) - 1
	if nextEndOff > (len(d.rendered) - 1) {
		d.endoff = (len(d.rendered) - 1)
	} else {
		d.endoff = nextEndOff
	}

	if !d.editingMode {
		renderedRowsNum := d.endoff - d.startoff + 1
		if d.cy > renderedRowsNum {
			d.cy = renderedRowsNum
		}
	}

	log.Default().Printf("looping from %d to %d\n", d.startoff, d.endoff)
	var printed int
	for i := d.startoff; i <= d.endoff; i++ {

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT && !d.editingMode {
			buf.WriteString(escape.REVERSE_COLOR)
			buf.WriteString(escape.SelectGraphicRendition(escape.WHITE))
		}

		// TODO check that the terminal supports Unicode output, before outputting a Unicode character
		// if so, the "LANG" env variable should contain "UTF"

		runes := utf8.RuneCountInString(string(d.rendered[i]))

		if runes > d.width {
			log.Default().Printf("runes for line %d exceed screen width: %d\n", i, runes)
			continue
		}

		for _, c := range string(d.rendered[i]) {
			buf.WriteRune(c)
		}

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT {
			buf.WriteString(escape.ATTRIBUTES_OFF)
			buf.WriteString(escape.DEFAULT_FG_COLOR)
		}

		buf.WriteString("\r\n")
		printed++
	}

	for ; printed < d.height-BOTTOM_PADDING; printed++ {
		buf.WriteString("\r\n")
	}

	for k := 0; k < d.width; k++ {
		buf.WriteString("-")
	}
	buf.WriteString("\r\n")

	if DebugMode {
		d.bottomRightCorner = fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.cy, d.cx, d.startoff, d.endoff, d.height, d.width)
	} else {
		d.bottomRightCorner = fmt.Sprintf("%d/%d", d.cy+d.startoff, len(d.rendered))
	}
	padding := d.width - utf8.RuneCountInString(d.bottomBarMsg) - 1

	buf.WriteString(escape.REVERSE_COLOR)
	buf.WriteString(escape.SelectGraphicRendition(d.bottomBarColor))
	buf.WriteString(fmt.Sprintf("%s %*s\r\n", d.bottomBarMsg, padding, d.bottomRightCorner))

	buf.WriteString(escape.ATTRIBUTES_OFF)
	buf.WriteString(escape.DEFAULT_FG_COLOR)
}
