package display

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/internal/ansi"
	"github.com/giulianopz/newscanoe/internal/app"
	"github.com/giulianopz/newscanoe/internal/ascii"
	"github.com/giulianopz/newscanoe/internal/cache"
	"github.com/giulianopz/newscanoe/internal/config"
	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/util"
	"github.com/giulianopz/newscanoe/internal/xterm"
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
	BOTTOM_PADDING = 2
)

// bottom bar messages
const (
	urlsListSectionMsg     = "HELP: q = quit | r = reload | R = reload all | a = add a feed"
	articlesListSectionMsg = "HELP: \u21B5 = view article | \u232B = go back"
	articleTextSectionMsg  = "HELP: \u232B = go back |  \u25B2 = scroll up | \u25BC = scroll down"
)

type cell struct {
	char   rune
	params []int
}

func newCell(c rune) *cell {
	return &cell{
		char: c,
	}
}

func fromString(s string) []*cell {
	cells := make([]*cell, 0)
	for _, r := range s {
		cells = append(cells, newCell(r))
	}
	return cells
}

func fromStringWithStyle(s string, params ...int) []*cell {
	cells := make([]*cell, 0)
	for _, r := range s {
		cells = append(cells, newCell(r).withStyle(params...))
	}
	if len(cells) != 0 {
		cells = append([]*cell{newCell(ascii.NULL).withStyle(params...)}, cells...)
		cells = append(cells, newCell(ascii.NULL).withStyle(ansi.ALL_ATTRIBUTES_OFF))
	}
	return cells
}

func (c *cell) withStyle(a ...int) *cell {
	c.params = append(make([]int, 0), a...)
	return c
}

func (c cell) String() string {
	if len(c.params) == 0 {
		return string(c.char)
	}
	if c.char == ascii.NULL {
		return ansi.SGR(c.params...)
	}
	return ansi.SGR(c.params...) + string(c.char)
}

func stringify(cells []*cell) string {
	var ret string
	for _, c := range cells {
		ret += c.String()
	}
	return ret
}

/*
display is the core struct handling the whole state of the application:
it draws the UI handling the displayed textual content as a window
which slides over a 2-dimensional array (raw and rendered)
enclosed by two bars displaying some navigation context to the user
----------------
|   top-bar    |
|______________|
|              |<-startoff
|              |
|  I           |<-(cx,cy)
|              |
|              |
|______________|<-endoff
|  bottom-bar  |
----------------
*/
type display struct {
	mu sync.Mutex

	// current position of cursor
	current *pos
	// previous positions of cursor
	previous []*pos

	// size of terminal window
	height int
	width  int

	// display raw text
	raw [][]byte
	// display rendered text
	rendered [][]*cell

	// YAML config
	config *config.Config
	// gob cache
	cache *cache.Cache

	// message displayed in the bottom bar
	topBarMsg string
	// message displayed in the bottom bar
	bottomBarMsg string
	// message displayed in the right corner of the bottom bar
	bottomRightCorner string

	ListenToKeyStrokes bool

	parser *feed.Parser

	editingMode bool
	editingBuf  []string

	currentSection    int
	currentFeedUrl    string
	currentArticleUrl string
}

type pos struct {
	// cursor's position within visible content window
	cx int
	cy int

	// offsets of rendered content window
	startoff int
	endoff   int
}

func (d *display) trackPos() {
	d.previous = append(d.previous, &pos{
		cx:       d.current.cx,
		cy:       d.current.cy,
		startoff: d.current.startoff,
		endoff:   d.current.endoff,
	})
}

func (d *display) restorePos() {
	if len(d.previous) != 0 {
		last := d.previous[len(d.previous)-1]
		d.current.cy, d.current.cx = last.cy, last.cx
		d.current.startoff, d.current.endoff = last.startoff, last.endoff

		d.previous[len(d.previous)-1] = nil
		d.previous = d.previous[:len(d.previous)-1]
	} else {
		d.current.cy, d.current.cx = 1, 1
		d.current.startoff, d.current.endoff = 0, 0
	}
}

func New() *display {

	d := &display{
		current: &pos{
			cx:       1,
			cy:       1,
			startoff: 0,
			endoff:   0,
		},
		previous:           make([]*pos, 0),
		ListenToKeyStrokes: true,
		config:             &config.Config{},
		cache:              cache.NewCache(),
		parser:             feed.NewParser(),
	}
	return d
}

func (d *display) setMaxEndOff() {
	d.current.endoff = d.current.startoff + d.getContentWindowLen() - 1
	if d.current.endoff > (len(d.rendered) - 1) {
		d.current.endoff = (len(d.rendered) - 1)
	}

	if !d.editingMode {
		max := d.current.endoff - d.current.startoff + 1
		if d.current.cy > max {
			d.current.cy = max
		}
	}
}

func (d *display) getContentWindowLen() int {
	return d.height - BOTTOM_PADDING - TOP_PADDING
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

	d.ListenToKeyStrokes = false

	// this is just what 'clear' does:
	// $ clear  | od -c
	// 0000000 033   [   H 033   [   2   J 033   [   3   J
	// 0000013
	// please, read here: https://superuser.com/questions/1667569/how-can-i-make-clear-preserve-entire-scrollback-buffer

	fmt.Fprint(os.Stdout, ansi.ShowCursor())
	fmt.Fprint(os.Stdout, ansi.MoveCursor(1, 1))
	fmt.Fprint(os.Stdout, ansi.Erase(ansi.ERASE_ENTIRE_SCREEN))
	fmt.Fprint(os.Stdout, ansi.Erase(xterm.CLEAR_SCROLLBACK_BUFFER))

	fmt.Fprintf(os.Stdout, xterm.DISABLE_BRACKETED_PASTE)

	quitC <- true
}

func (d *display) appendToRaw(s string) {
	d.raw = append(d.raw, []byte(s))
}

func (d *display) appendToRendered(cells []*cell) {
	d.rendered = append(d.rendered, cells)
}

func (d *display) currentRow() int {
	return d.current.cy - 1 + d.current.startoff
}

func (d *display) currentUrl() string {
	return string(d.raw[d.currentRow()])
}

func (d *display) resetCurrentPos() {
	d.current.cy = 1
	d.current.cx = 1
	d.current.startoff = 0
}

func (d *display) resetRows() {
	d.raw = make([][]byte, 0)
	d.rendered = make([][]*cell, 0)
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

func (d *display) LoadConfig() error {
	configFile, err := util.GetConfigFilePath()
	if err != nil {
		return err
	}

	if util.Exists(configFile) {
		if err := d.config.Decode(configFile); err != nil {
			return err
		}
	}
	return nil
}

func (d *display) LoadCache() error {
	cachePath, err := util.GetCacheFilePath()
	if err != nil {
		return err
	}

	if util.Exists(cachePath) {
		if err := d.cache.Decode(cachePath); err != nil {
			return err
		}
	}

	d.cache.Merge(d.config)
	return nil
}

func (d *display) exitEditingMode() {
	d.editingMode = false
	d.editingBuf = []string{}
}

func (d *display) enterEditingMode() {
	log.Default().Println("live editing enabled")

	d.editingMode = true
	d.editingBuf = []string{}

	d.current.cy = d.height
	d.current.cx = 1

	d.setBottomMessage("")
}

func (d *display) draw(buf *bytes.Buffer) {

	/* top bar */

	write(buf, ansi.SGR(ansi.REVERSE_COLOR), "cannot reverse color")

	padding := d.width - utf8.RuneCountInString(app.Name) - utf8.RuneCountInString(d.topBarMsg) - 2
	log.Default().Printf("top-padding: %d", padding)
	if padding > 0 {
		write(buf, fmt.Sprintf("%s %s %*s\r\n", app.Name, d.topBarMsg, padding, app.Version), "cannot write top bar")
	} else {
		write(buf, app.Name, "cannot write top bar")

		padding = d.width - utf8.RuneCountInString(app.Name) - utf8.RuneCountInString(app.Version)
		for i := padding; i > 0; i-- {
			write(buf, " ", "cannot write empty char")
		}
		write(buf, fmt.Sprintf("%s\r\n", app.Version), "cannot write app version")
	}

	write(buf, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF), "cannot write reset escape sequence")

	/* main content */

	for k := 0; k < d.width; k++ {
		write(buf, "-", "cannot write hyphen")
	}
	write(buf, "\r\n", "cannot write carriage return")

	d.setMaxEndOff()

	log.Default().Printf("looping from %d to %d: %d\n", d.current.startoff, d.current.endoff, d.current.endoff-d.current.startoff)
	var printed int
	for i := d.current.startoff; i <= d.current.endoff; i++ {

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT && !d.editingMode {
			write(buf, ansi.SGR(ansi.REVERSE_COLOR), "cannot reverse color")
		}

		row := d.rendered[i]

		var runes int
		for _, c := range row {
			if c.char != ascii.NULL {
				runes++
			}
		}

		if runes > d.width {
			log.Default().Printf("current line length %d exceeds screen width: %d\n", runes, d.width)
			row = row[:d.width]
		}

		line := stringify(row)
		if line == "" {
			log.Default().Println("found empty row")
			line = " "
		}
		log.Default().Printf("writing to buf line #%d: %q\n", i, line)

		write(buf, ansi.WhiteFG(), "cannot write escape for white foreground")
		write(buf, line, "cannot write line")

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT {
			write(buf, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF), "cannot write reset escape sequence")
		}

		write(buf, "\r\n", "cannot write carriage return")

		printed++
	}

	for ; printed < d.height-BOTTOM_PADDING-TOP_PADDING; printed++ {
		write(buf, "\r\n", "cannot write carriage return")
	}

	/* bottom bar */

	for k := 0; k < d.width; k++ {
		write(buf, "-", "cannot write hyphen")
	}
	write(buf, "\r\n", "cannot write carriage return")

	write(buf, ansi.SGR(ansi.REVERSE_COLOR), "cannot reverse color")

	d.bottomRightCorner = fmt.Sprintf("%d/%d", d.current.cy+d.current.startoff, len(d.rendered))
	if DebugMode {
		d.bottomRightCorner = fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.current.cy, d.current.cx, d.current.startoff, d.current.endoff, d.height, d.width)
	}

	padding = d.width - utf8.RuneCountInString(d.bottomBarMsg) - 1
	log.Default().Printf("bottom-padding: %d", padding)

	if padding > 0 {
		write(buf, fmt.Sprintf("%s %*s", d.bottomBarMsg, padding, d.bottomRightCorner), "cannot write bottom right corner text")
	} else {
		padding = d.width - utf8.RuneCountInString(d.bottomRightCorner)
		for i := padding; i > 0; i-- {
			write(buf, " ", "cannot write empty string")
		}
		write(buf, d.bottomRightCorner, "cannot write bottom right corner text")
	}
	write(buf, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF), "cannot reset display attributes")
}

func write(buf *bytes.Buffer, s, msg string) {
	if _, err := buf.WriteString(s); err != nil {
		log.Default().Printf("%s: %v", msg, err)
	}
}

func (d *display) RefreshScreen() {

	log.Default().Println("refreshing screen")

	buf := &bytes.Buffer{}

	buf.WriteString(ansi.HideCursor())
	buf.WriteString(ansi.MoveCursor(1, 1))

	buf.WriteString(ansi.Erase(ansi.ERASE_ENTIRE_SCREEN))
	buf.WriteString(ansi.Erase(xterm.CLEAR_SCROLLBACK_BUFFER))

	switch d.currentSection {

	case URLS_LIST:
		d.renderFeedList()

	case ARTICLES_LIST:
		d.renderArticleList()

	case ARTICLE_TEXT:
		d.renderArticleText()
	}

	d.draw(buf)

	buf.WriteString(ansi.MoveCursor(d.current.cy, d.current.cx))
	if d.editingMode {
		buf.WriteString(ansi.ShowCursor())
	}

	fmt.Fprint(os.Stdout, buf)
}
