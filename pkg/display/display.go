package display

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/pkg/ansi"
	"github.com/giulianopz/newscanoe/pkg/app"
	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/giulianopz/newscanoe/pkg/xterm"
	"github.com/mmcdole/gofeed"
)

var DebugMode bool

// display sections
const (
	URLS_LIST = iota
	ARTICLES_LIST
	ARTICLE_TEXT
)

// num of lines reserved to top and bottom bars plus a final empty row
const (
	TOP_PADDING    = 2
	BOTTOM_PADDING = 3
)

// bottom bar messages
const (
	urlsListSectionMsg     = "HELP: q = quit | r = reload | R = reload all | a = add a feed"
	articlesListSectionMsg = "HELP: \u21B5 = view article | \u232B = go back"
	articleTextSectionMsg  = "HELP: \u232B = go back |  \u25B2 = scroll up | \u25BC = scroll down"
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

	// color of top and bottom bars
	barsColor int
	// message displayed in the bottom bar
	topBarMsg string
	// message displayed in the bottom bar
	bottomBarMsg string
	// message displayed in the right corner of the bottom bar
	bottomRightCorner string

	mu sync.Mutex

	// display raw text
	raw [][]byte
	// display rendered text
	rendered [][]rune
	// gob cache
	cache *cache.Cache

	editingMode bool
	editingBuf  []string

	currentSection    int
	currentArticleUrl string
	currentFeedUrl    string

	ListenToKeyStroke bool

	client *http.Client

	parser *gofeed.Parser
}

func New() *display {

	d := &display{
		cx:                1,
		cy:                1,
		startoff:          0,
		endoff:            0,
		cache:             cache.NewCache(),
		barsColor:         ansi.WHITE,
		ListenToKeyStroke: true,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		parser: gofeed.NewParser(),
	}
	return d
}

func (d *display) setTopMessage(msg string) {
	if utf8.RuneCountInString(msg) < (d.width - utf8.RuneCountInString(app.Name) - utf8.RuneCountInString(app.Version) - 2) {
		d.topBarMsg = msg
	}
}

func (d *display) setBottomMessage(msg string) {
	d.bottomBarMsg = msg
}

func (d *display) setTmpBottomMessage(t time.Duration, msg string) {
	previous := d.bottomBarMsg
	d.setBottomMessage(msg)
	go func() {
		time.AfterFunc(t, func() {
			d.barsColor = ansi.WHITE
			d.setBottomMessage(previous)
		})
	}()
}

func (d *display) SetWindowSize(w, h int) {
	log.Default().Println("resetting window size")
	d.width = w
	d.height = h
}

func (d *display) Quit(quitC chan bool) {

	log.Default().Println("quitting")

	d.ListenToKeyStroke = false

	fmt.Fprint(os.Stdout, ansi.ShowCursor())
	fmt.Fprint(os.Stdout, ansi.Erase(ansi.ERASE_ENTIRE_SCREEN))
	fmt.Fprint(os.Stdout, ansi.MoveCursor(1, 1))

	fmt.Fprint(os.Stdout, xterm.CLEAR_SCROLLBACK_BUFFER)
	fmt.Fprintf(os.Stdout, xterm.DISABLE_BRACKETED_PASTE)

	quitC <- true
}

func (d *display) appendToRaw(s string) {
	d.raw = append(d.raw, []byte(s))
}

func (d *display) appendToRendered(s string) {
	d.rendered = append(d.rendered, []rune(s))
}

func (d *display) currentRow() int {
	return d.cy - 1 + d.startoff
}

func (d *display) currentUrl() string {
	return string(d.raw[d.currentRow()])
}

func (d *display) resetCoordinates() {
	d.cy = 1
	d.cx = 1
	d.startoff = 0
}

func (d *display) resetRows() {
	d.raw = make([][]byte, 0)
	d.rendered = make([][]rune, 0)
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

func (d *display) LoadCache() error {
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
	d.barsColor = color
}

func (d *display) enterEditingMode() {
	log.Default().Println("live editing enabled")

	d.editingMode = true
	d.editingBuf = []string{}

	d.cy = d.height - 1
	d.cx = 1

	d.setBottomMessage("")
}

func (d *display) canBeParsed(url string) bool {
	if _, err := d.parser.ParseURL(url); err != nil {
		log.Default().Printf("cannot parse feed url: %v\n", err)
		return false
	}
	return true
}

func (d *display) draw(buf *bytes.Buffer) {

	buf.WriteString(ansi.SGR(ansi.REVERSE_COLOR))
	buf.WriteString(ansi.SGR(d.barsColor))

	padding := d.width - utf8.RuneCountInString(app.Name) - utf8.RuneCountInString(d.topBarMsg) - 2
	log.Default().Printf("top-padding: %d", padding)
	if padding > 0 {
		buf.WriteString(fmt.Sprintf("%s %s %*s\r\n", app.Name, d.topBarMsg, padding, app.Version))
	} else {
		buf.WriteString(app.Name)
		padding = d.width - utf8.RuneCountInString(app.Name) - utf8.RuneCountInString(app.Version)
		for i := padding; i > 0; i-- {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%s\r\n", app.Version))
	}

	buf.WriteString(ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))
	buf.WriteString(ansi.SGR(ansi.DEFAULT_FG_COLOR))

	for k := 0; k < d.width; k++ {
		buf.WriteString("-")
	}

	nextEndOff := d.startoff + (d.height - BOTTOM_PADDING - TOP_PADDING) - 1
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

	log.Default().Printf("looping from %d to %d: %d\n", d.startoff, d.endoff, d.endoff-d.startoff)
	var printed int
	for i := d.startoff; i <= d.endoff; i++ {

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT && !d.editingMode {
			buf.WriteString(ansi.SGR(ansi.REVERSE_COLOR))
			buf.WriteString(ansi.SGR(ansi.WHITE))
		}

		// TODO check that the terminal supports Unicode output, before outputting a Unicode character
		// if so, the "LANG" env variable should contain "UTF"

		line := string(d.rendered[i])
		if line == "" {
			line = " "
		} else {
			runes := utf8.RuneCountInString(line)
			if runes > d.width {
				log.Default().Printf("truncating current line because its length %d exceeda screen width: %d\n", i, runes)
				line = util.Truncate(line, d.width)
			}
		}

		log.Default().Printf("writing to buf line #%d: %q\n", i, line)

		_, err := buf.Write([]byte(line))
		if err != nil {
			log.Default().Printf("cannot write byte array %q: %v", []byte(" "), err)
		}

		buf.WriteString("\r\n")

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT {
			buf.WriteString(ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))
			buf.WriteString(ansi.SGR(ansi.DEFAULT_FG_COLOR))
		}

		printed++
	}

	for ; printed < d.height-BOTTOM_PADDING-TOP_PADDING; printed++ {
		buf.WriteString("\r\n")
	}

	for k := 0; k < d.width; k++ {
		buf.WriteString("-")
	}
	buf.WriteString("\r\n")

	d.bottomRightCorner = fmt.Sprintf("%d/%d", d.cy+d.startoff, len(d.rendered))
	if DebugMode {
		d.bottomRightCorner = fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.cy, d.cx, d.startoff, d.endoff, d.height, d.width)
	}

	padding = d.width - utf8.RuneCountInString(d.bottomBarMsg) - 1
	log.Default().Printf("bottom-padding: %d", padding)

	buf.WriteString(ansi.SGR(ansi.REVERSE_COLOR))
	buf.WriteString(ansi.SGR(d.barsColor))

	if padding > 0 {
		buf.WriteString(fmt.Sprintf("%s %*s", d.bottomBarMsg, padding, d.bottomRightCorner))
	} else {
		padding = d.width - utf8.RuneCountInString(d.bottomRightCorner)
		for i := padding; i > 0; i-- {
			buf.WriteString(" ")
		}
		buf.WriteString(d.bottomRightCorner)
	}

	buf.WriteString(ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))
	buf.WriteString(ansi.SGR(ansi.DEFAULT_FG_COLOR))
}

func (d *display) RefreshScreen() {

	log.Default().Println("refreshing screen")

	buf := &bytes.Buffer{}

	buf.WriteString(ansi.Erase(ansi.ERASE_ENTIRE_SCREEN))
	buf.WriteString(ansi.HideCursor())
	buf.WriteString(ansi.MoveCursor(1, 1))

	switch d.currentSection {

	case URLS_LIST:
		d.renderURLs()

	case ARTICLES_LIST:
		d.renderArticlesList()

	case ARTICLE_TEXT:
		d.renderArticleText()
	}

	d.draw(buf)

	buf.WriteString(ansi.MoveCursor(d.cy, d.cx))
	if d.editingMode {
		buf.WriteString(ansi.ShowCursor())
	}

	fmt.Fprint(os.Stdout, buf)
}
