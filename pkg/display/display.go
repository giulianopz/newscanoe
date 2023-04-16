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

	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/escape"
	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
)

var DebugMode bool

// display sections
const (
	URLS_LIST = iota
	ARTICLES_LIST
	ARTICLE_TEXT
)

// num of lines reserved to bottom bar plus final empty row
const BOTTOM_PADDING = 3

// bottom bar messages
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

	client *http.Client

	parser *gofeed.Parser
}

func New() *display {

	d := &display{
		cx:             1,
		cy:             1,
		startoff:       0,
		endoff:         0,
		cache:          cache.NewCache(),
		bottomBarColor: escape.WHITE,
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		parser: gofeed.NewParser(),
	}
	return d
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

func (d *display) SetWindowSize(w, h int) {
	log.Default().Println("resetting window size")
	d.width = w
	d.height = h
}

func (d *display) Quit(quitC chan bool) {

	log.Default().Println("quitting")
	d.Quitting = true

	fmt.Fprint(os.Stdout, escape.SHOW_CURSOR)
	fmt.Fprint(os.Stdout, escape.ERASE_ENTIRE_SCREEN)
	fmt.Fprint(os.Stdout, escape.MoveCursor(1, 1))
	quitC <- true
}

func (d *display) appendToRaw(s string) {
	d.raw = append(d.raw, []byte(s))
}

func (d *display) appendToRendered(s string) {
	d.rendered = append(d.rendered, []byte(s))
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
	d.raw = make([][]byte, 0, 0)
	d.rendered = make([][]byte, 0, 0)
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

func (d *display) canBeParsed(url string) bool {
	if _, err := d.parser.ParseURL(url); err != nil {
		log.Default().Printf("cannot parse feed url: %v\n", err)
		return false
	}
	return true
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

func (d *display) RefreshScreen() {

	log.Default().Println("refreshing screen")

	buf := &bytes.Buffer{}

	buf.WriteString(escape.ERASE_ENTIRE_SCREEN)
	buf.WriteString(escape.HIDE_CURSOR)
	buf.WriteString(escape.MoveCursor(1, 1))

	if d.currentSection == ARTICLE_TEXT {
		d.renderText()
	}

	d.draw(buf)

	buf.WriteString(escape.MoveCursor(d.cy, d.cx))
	if d.editingMode {
		buf.WriteString(escape.SHOW_CURSOR)
	}

	fmt.Fprint(os.Stdout, buf)
}

func (d *display) renderText() {

	log.Default().Println("width: ", d.width)

	chars := make([]byte, 0)
	for row := range d.raw {
		if len(d.raw[row]) == 0 {
			chars = append(chars, '\n')
		}
		for _, c := range d.raw[row] {
			chars = append(chars, c)
		}
	}

	d.rendered = make([][]byte, 0)
	line := make([]byte, 0)
	for _, c := range chars {

		if c == '\r' || c == '\n' {

			if len(line) != 0 {
				d.rendered = append(d.rendered, line)
			}
			d.rendered = append(d.rendered, []byte{})
			line = make([]byte, 0)
			continue
		}

		if c == '\t' {
			for i := 0; i < 4; i++ {
				line = append(line, ' ')
			}
			continue
		}

		if len(line) < d.width-1 {
			line = append(line, c)
		} else {
			d.rendered = append(d.rendered, line)
			line = make([]byte, 0)
			line = append(line, c)
		}
	}

	if len(line) != 0 {
		d.rendered = append(d.rendered, line)
	}
}
