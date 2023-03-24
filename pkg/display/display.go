package display

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/pkg/cache"
	"github.com/giulianopz/newscanoe/pkg/termios"
	"github.com/giulianopz/newscanoe/pkg/util"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sys/unix"
)

var bottomPadding int = 3

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
)

type Display struct {
	// cursor's position within terminal coordinates
	cx int
	cy int
	// limits of rendered window
	startoff int
	endoff   int
	// window size
	height int
	width  int

	msgtatus string
	infotime time.Time

	origTermios unix.Termios

	rows     [][]byte
	rendered [][]byte
	cache    cache.Cache

	//TODO show pages coordinates
	pages          int
	currentPage    int
	currentSection int
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
	d.SetStatusMessage("HELP: Ctrl-Q = quit | Ctrl-r = reload | Ctrl-R = reload all")

	if err := d.LoadURLs(); err != nil {
		log.Fatal(err)
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
			d.cx = len(d.rendered[d.cy-1])
		}
	case ARROW_RIGHT:
		if (d.cx - 1) < (len(d.rendered[d.cy-1]) - 1) {
			d.cx++
		} else if d.cy >= 1 && d.cy < (d.height-bottomPadding) {
			d.cy++
			d.cx = 1
		}
	case ARROW_DOWN:
		if d.cy < (d.height - bottomPadding) {
			if (d.cx - 1) <= (len(d.rendered[d.cy]) - 1) {
				d.cy++
			}
		} else if d.endoff < len(d.rendered)-1 {
			d.startoff++
		}
	case ARROW_UP:
		if d.cy > 1 {
			if (d.cx - 1) <= (len(d.rendered[d.cy-2]) - 1) {
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
			log.Fatal(err)
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
		d.Reload(string(d.rows[d.cy-1+d.startoff]))
		d.RefreshScreen()

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.MoveCursor(input)

	case ENTER:
		d.LoadArticles(string(d.rows[d.cy-1+d.startoff]))

	case BACKSPACE:
		{
			switch d.currentSection {
			case ARTICLES_LIST:
				d.LoadURLs()
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
	d.msgtatus = msg
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

func (d *Display) LoadURLs() error {

	d.rows = make([][]byte, 0)
	d.rendered = make([][]byte, 0)

	if len(d.cache.Feeds) != 0 {

		for _, f := range d.cache.Feeds {
			d.rows = append(d.rows, []byte(f.Url))
			d.rendered = append(d.rendered, []byte(f.Title))
		}
	} else {
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

		fscanner := bufio.NewScanner(file)
		for fscanner.Scan() {

			url := fscanner.Bytes()
			if !strings.Contains(string(url), "#") {
				d.rows = append(d.rows, url)
				d.rendered = append(d.rendered, url)
				d.cache.Feeds = append(d.cache.Feeds, &cache.Feed{
					Url:   string(url),
					Title: string(url),
				})
			}
		}
	}

	d.currentSection = URLS_LIST
	return nil
}

var nonAlphaNumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func (d *Display) Reload(url string) {

	fp := gofeed.NewParser()
	parsedFeed, err := fp.ParseURL(url)
	if err != nil {
		// TODO
		panic(err)
	}

	title := nonAlphaNumericRegex.ReplaceAllString(parsedFeed.Title, "")

	for _, cachedFeed := range d.cache.Feeds {
		if cachedFeed.Url == url {
			cachedFeed.Title = title
			cachedFeed.Items = make([]*cache.Item, 0)

			for _, parsedItem := range parsedFeed.Items {
				cachedItem := &cache.Item{
					Title:   parsedItem.Title,
					Link:    parsedItem.Link,
					PubDate: *parsedItem.PublishedParsed,
				}
				cachedFeed.Items = append(cachedFeed.Items, cachedItem)
			}
		}
	}

	d.rendered[d.cy-1+d.startoff] = []byte(title)
}

func (d *Display) LoadArticles(url string) {

	for _, f := range d.cache.Feeds {
		if f.Url == url {

			d.rows = make([][]byte, 0)
			d.rendered = make([][]byte, 0)
			for _, i := range f.Items {
				d.rows = append(d.rows, []byte(i.Link))

				dateAndArticleName := fmt.Sprintf("%-20s %s", i.PubDate.Format("2006-January-02"), i.Title)
				d.rendered = append(d.rendered, []byte(dateAndArticleName))
			}
		}
	}
	d.currentSection = ARTICLES_LIST
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
			//TODO handle rows longer than screen's width
			if j < (d.width) {
				buf.WriteString(string(d.rendered[i][j]))
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

	// TODO enable only if --debug falg is set to true
	tracking := fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.cy, d.cx, d.startoff, d.endoff, d.height, d.width)

	buf.WriteString(fmt.Sprintf("%s %135s\r\n", d.msgtatus, tracking))
}
