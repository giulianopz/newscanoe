package display

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/internal/ansi"
	"github.com/giulianopz/newscanoe/internal/app"
	"github.com/giulianopz/newscanoe/internal/bar"
	"github.com/giulianopz/newscanoe/internal/cache"
	"github.com/giulianopz/newscanoe/internal/config"
	"github.com/giulianopz/newscanoe/internal/feed"
	"github.com/giulianopz/newscanoe/internal/util"
	"github.com/giulianopz/newscanoe/internal/xterm"
)

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
// for Unicode codes, see: http://xahlee.info/comp/unicode_computing_symbols.html
// some of them are not correctly rendered by gnome-terminal: https://gitlab.gnome.org/GNOME/vte/-/issues/2580
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
		cells = append([]*cell{newCell(NULL).withStyle(params...)}, cells...)
		cells = append(cells, newCell(NULL).withStyle(ansi.ALL_ATTRIBUTES_OFF))
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
	if c.char == NULL {
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
	debugMode bool

	QuitC chan bool

	mu sync.Mutex

	// current position of cursor
	current *pos
	// previous positions of cursor
	previous []*pos

	hideCursor bool

	// size of terminal window
	height int
	width  int

	// display raw text
	raw [][]byte
	// display rendered text
	rendered [][]*cell

	// config
	config *config.Config
	// gob cache
	cache *cache.Cache

	// message displayed in the bottom bar
	topBarMsg string
	// message displayed in the bottom bar
	bottomBarMsg string

	parser *feed.Parser

	editingMode bool
	editingBuf  *buffer

	currentSection    int
	currentFeedUrl    string
	currentArticleUrl string
}

type pos struct {
	// cursor's position within visible content window
	cy, cx int
	// offsets of rendered content window
	startoff, endoff int
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

func New(debugMode bool) *display {

	d := &display{
		debugMode: debugMode,
		QuitC:     make(chan bool, 1),
		current: &pos{
			cx:       1,
			cy:       1,
			startoff: 0,
			endoff:   0,
		},
		previous: make([]*pos, 0),
		config:   &config.Config{},
		cache:    cache.NewCache(),
		parser:   feed.NewParser(),
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

	previousMsg := d.bottomBarMsg
	d.setBottomMessage(msg)
	d.hideCursor = true

	go func() {
		time.AfterFunc(t, func() {

			defer func() {
				d.setBottomMessage(previousMsg)
				d.hideCursor = false
			}()

			fmt.Fprint(os.Stdout, ansi.MoveCursor(d.height, 1))
			fmt.Fprint(os.Stdout, ansi.EraseToEndOfLine(ansi.ERASE_ENTIRE_LINE))
			fmt.Fprint(os.Stdout, ansi.SGR(ansi.REVERSE_COLOR))
			fmt.Fprint(os.Stdout, util.PadToRight(previousMsg, d.width))
			fmt.Fprint(os.Stdout, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))
		})
	}()
}

func (d *display) SetWindowSize(w, h int) {
	log.Default().Println("resetting window size")
	d.width = w
	d.height = h
}

// Clear does what clear(1) does:
// $ clear  | od -c
// 0000000 033   [   H 033   [   2   J 033   [   3   J
// 0000013
// ref: https://superuser.com/questions/1667569/how-can-i-make-clear-preserve-entire-scrollback-buffer
// Then it restores the terminal's previous settings
func (d *display) Clear() {
	log.Default().Println("clearing screen")
	fmt.Fprint(os.Stdout, ansi.ShowCursor())
	fmt.Fprint(os.Stdout, ansi.MoveCursor(1, 1))
	fmt.Fprint(os.Stdout, ansi.EraseToEndOfScreen(ansi.ERASE_ENTIRE_SCREEN))
	fmt.Fprint(os.Stdout, ansi.EraseToEndOfScreen(xterm.CLEAR_SCROLLBACK_BUFFER))
	fmt.Fprintf(os.Stdout, xterm.DISABLE_BRACKETED_PASTE)
}

func (d *display) appendToRaw(s string) {
	d.raw = append(d.raw, []byte(s))
}

func (d *display) indexOf(url string) int {
	sort.SliceStable(d.config.Feeds, func(i, j int) bool {
		return strings.ToLower(d.config.Feeds[i].Name) < strings.ToLower(d.config.Feeds[j].Name)
	})
	for idx, f := range d.config.Feeds {
		if f.Url == url {
			return idx
		}
	}
	return -1
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
	d.editingBuf = nil
}

func (d *display) enterEditingMode() {
	log.Default().Println("live editing enabled")

	d.editingMode = true
	d.editingBuf = new(buffer)

	d.current.cy = d.height
	d.current.cx = 1

	d.setBottomMessage("")
}

func (d *display) draw(buf *bytes.Buffer) {

	/* top bar */

	topBar := bar.NewBar()
	if d.topBarMsg != "" {
		topBar.SetText(app.Name+" "+d.topBarMsg, app.Version)
	} else {
		topBar.SetText(app.Name, app.Version)
	}
	fmt.Fprint(buf, topBar.Build(d.width))
	fmt.Fprint(buf, "\r\n")

	fmt.Fprint(buf, util.LineOf(d.width, "\u2500"))
	fmt.Fprint(buf, "\r\n")

	/* main content */

	d.setMaxEndOff()

	log.Default().Printf("looping from %d to %d: %d\n", d.current.startoff, d.current.endoff, d.current.endoff-d.current.startoff)
	var printed int
	for i := d.current.startoff; i <= d.current.endoff; i++ {

		var runes int

		if d.currentSection != ARTICLE_TEXT {
			var arrow string
			if i != d.height && i == d.currentRow() {
				// https://en.wikipedia.org/wiki/Geometric_Shapes_(Unicode_block)
				arrow = "\u25B6"
			}
			fmt.Fprintf(buf, "%-2s", arrow)

			runes += 2
		}

		row := d.rendered[i]

		for _, c := range row {
			if c.char != NULL {
				runes++
			}
		}

		if runes > d.width {
			log.Default().Printf("current line length %d exceeds screen width: %d\n", runes, d.width)
			row = row[:len(row)-(runes-d.width+1)]
		}

		line := stringify(row)
		if line == "" {
			log.Default().Println("found empty row")
			line = " "
		}

		log.Default().Printf("writing to buf line #%d: %q\n", i, line)

		fmt.Fprint(buf, ansi.WhiteFG())
		fmt.Fprint(buf, line)
		fmt.Fprint(buf, "\r\n")

		printed++
	}

	for ; printed < d.height-BOTTOM_PADDING-TOP_PADDING; printed++ {
		fmt.Fprint(buf, "\r\n")
	}

	/* bottom bar */

	fmt.Fprint(buf, util.LineOf(d.width, "\u2500"))
	fmt.Fprint(buf, "\r\n")

	bottomBar := bar.NewBar()
	switch {
	case d.editingMode:
		bottomBar.SetText(d.bottomBarMsg, "")
	case d.debugMode:
		bottomBar.SetText(d.bottomBarMsg, fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.current.cy, d.current.cx, d.current.startoff, d.current.endoff, d.height, d.width))
	default:
		bottomBar.SetText(d.bottomBarMsg, fmt.Sprintf("%d/%d", d.current.cy+d.current.startoff, len(d.rendered)))
	}

	fmt.Fprint(buf, bottomBar.Build(d.width))
}

func (d *display) RefreshScreen() {

	log.Default().Println("refreshing screen")

	buf := &bytes.Buffer{}

	fmt.Fprint(buf, ansi.HideCursor())
	fmt.Fprint(buf, ansi.MoveCursor(1, 1))

	fmt.Fprint(buf, ansi.EraseToEndOfScreen(ansi.ERASE_ENTIRE_SCREEN))
	fmt.Fprint(buf, ansi.EraseToEndOfScreen(xterm.CLEAR_SCROLLBACK_BUFFER))

	d.draw(buf)

	fmt.Fprint(buf, ansi.MoveCursor(d.current.cy, d.current.cx))
	if d.editingMode && !d.hideCursor {
		fmt.Fprint(buf, ansi.ShowCursor())
	}

	fmt.Fprint(os.Stdout, buf.String())
}

func (d *display) ListenToInput() {
	for {
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Default().Printf("recover from: %v\nstack trace: %v\n", r, string(debug.Stack()))
					d.setTmpBottomMessage(2*time.Second, "something bad happened: check the logs")
				}
			}()

			d.RefreshScreen()
			input := d.ReadKeyStroke(os.Stdin.Fd())
			d.ProcessKeyStroke(input)
		}()
	}
}
