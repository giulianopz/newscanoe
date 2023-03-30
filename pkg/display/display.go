package display

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

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
	BOTTOM_PADDING = 3
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

	bottomBarMsg string

	//infotime  time.Time

	origTermios unix.Termios

	rows     [][]byte
	rendered [][]byte
	cache    *cache.Cache

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
		cache:    cache.NewCache(),
	}

	d.SetWindowSize(in)

	if err := d.LoadCache(); err != nil {
		log.Panicln(err)
	}

	if err := d.LoadURLs(); err != nil {
		log.Panicln(err)
	}

	return d
}

func (d *Display) currentRow() int {
	return d.cy - 1 + d.startoff
}

func (d *Display) resetCoordinates() {
	d.cy = 1
	d.cx = 1
	d.startoff = 0
}

func (d *Display) resetRows() {
	d.rows = make([][]byte, 0)
	d.rendered = make([][]byte, 0)
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
			d.cx = len(d.rendered[d.currentRow()])
		}
	case ARROW_RIGHT:
		if (d.cx - 1) < (len(d.rendered[d.currentRow()]) - 1) {
			d.cx++
		} else if d.cy >= 1 && d.cy < (d.height-BOTTOM_PADDING) {
			d.cy++
			d.cx = 1
		}
	case ARROW_DOWN:
		if d.cy < (d.height - BOTTOM_PADDING) {
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
			//TODO use new slog
			log.Default().Println(err)
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
	fmt.Fprint(os.Stdout, "\x1b[?25h")
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
			d.LoadFeed(string(d.rows[d.currentRow()]))
		}

	case ctrlPlus('R'), 'R':
		if d.currentSection == URLS_LIST {
			d.LoadAllFeeds()
		}

	case ctrlPlus('o'), 'o':
		if d.currentSection == ARTICLES_LIST {
			if !headless() {
				d.openWithBrowser(string(d.rows[d.currentRow()]))
			}
		}
	case ctrlPlus('l'), 'l':
		if d.currentSection == ARTICLES_LIST {
			if isLynxPresent() {
				d.openWithLynx(string(d.rows[d.currentRow()]))
			}
		}

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.MoveCursor(input)

	case ENTER:
		{
			switch d.currentSection {
			case URLS_LIST:
				d.LoadArticlesList(string(d.rows[d.currentRow()]))
			case ARTICLES_LIST:
				d.LoadArticleText(string(d.rows[d.currentRow()]))
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
	//buf.WriteString("\x1b[?25h")

	fmt.Fprint(os.Stdout, buf)
}

func (d *Display) SetBottomMessage(msg string) {
	d.bottomBarMsg = msg
}

func (d *Display) SetTmpBottomMessage(t time.Duration, msg string) {
	previous := d.bottomBarMsg
	d.SetBottomMessage(msg)
	go func() {
		time.AfterFunc(t, func() {
			d.SetBottomMessage(previous)
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

	cached := make(map[string]bool, 0)

	for _, cachedFeed := range d.cache.GetFeeds() {
		cached[cachedFeed.Url] = true
		d.rows = append(d.rows, []byte(cachedFeed.Url))
		d.rendered = append(d.rendered, []byte(cachedFeed.Title))
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {

		url := scanner.Bytes()
		if !strings.Contains(string(url), "#") {

			if _, present := cached[string(url)]; !present {

				d.rows = append(d.rows, url)
				d.rendered = append(d.rendered, url)
				d.cache.AddFeedUrl(string(url))
			}
		}
	}

	d.cy = 1
	d.cx = 1
	d.currentSection = URLS_LIST

	d.SetBottomMessage("HELP: Ctrl-q/q = quit | Ctrl-r/r = reload | Ctrl-R/R = reload all")

	return nil
}

func (d *Display) LoadFeed(url string) {

	fp := gofeed.NewParser()
	parsedFeed, err := fp.ParseURL(url)
	if err != nil {
		log.Default().Println(err)
	}

	title := strings.TrimSpace(parsedFeed.Title)

	if err := d.cache.AddFeed(parsedFeed, url); err != nil {
		log.Default().Println(err)
		d.SetTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
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

func (d *Display) LoadAllFeeds() {

	origMsg := d.bottomBarMsg

	for id := range d.rows {

		url := string(d.rows[id])

		log.Default().Printf("loading feed #%d from url %s\n", id, url)

		fp := gofeed.NewParser()
		parsedFeed, err := fp.ParseURL(url)
		if err != nil {
			log.Default().Println(err)
		}

		title := strings.TrimSpace(parsedFeed.Title)

		if err := d.cache.AddFeed(parsedFeed, url); err != nil {
			log.Default().Println(err)
			d.SetTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
			return
		}

		d.rendered[id] = []byte(title)

		d.SetBottomMessage(fmt.Sprintf("loading all feeds, please wait........%d/%d", id+1, len(d.rows)))
		d.RefreshScreen()
	}

	d.SetBottomMessage(origMsg)

	go func() {
		if err := d.cache.Encode(); err != nil {
			log.Default().Println(err.Error())
		}
	}()

}

func (d *Display) LoadArticlesList(url string) {

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == url {

			if len(cachedFeed.Items) == 0 {
				d.SetTmpBottomMessage(3*time.Second, "feed not yet loaded: press r!")
				return
			}

			d.resetRows()

			for _, item := range cachedFeed.Items {
				d.rows = append(d.rows, []byte(item.Url))
				d.rendered = append(d.rendered, []byte(util.RenderArticleRow(item.PubDate, item.Title)))
			}

			d.resetCoordinates()

			d.currentSection = ARTICLES_LIST
			d.currentFeedUrl = url

			var browserHelp string
			if !headless() {
				browserHelp = " | Ctrl-o/o = open with browser"
			}

			var lynxHelp string
			if isLynxPresent() {
				browserHelp = " | Ctrl-l/l = open with lynx"
			}

			d.SetBottomMessage(fmt.Sprintf("HELP: Enter = view article | Backspace = go back %s %s", browserHelp, lynxHelp))
		}
	}
}

func (d *Display) LoadArticleText(url string) {

	for _, cachedFeed := range d.cache.GetFeeds() {
		if cachedFeed.Url == d.currentFeedUrl {

			for _, i := range cachedFeed.Items {

				if i.Url == url {

					resp, err := http.Get(i.Url)
					if err != nil {
						log.Default().Println(err)
						d.SetTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load article from url: %s", url))
						return
					}
					defer resp.Body.Close()

					converter := md.NewConverter("", true, nil)
					markdown, err := converter.ConvertReader(resp.Body)
					if err != nil {
						log.Default().Println(err)
						d.SetTmpBottomMessage(3*time.Second, "cannot parse article text!")
						return
					}

					d.resetRows()

					scanner := bufio.NewScanner(&markdown)
					for scanner.Scan() {

						line := strings.TrimSpace(scanner.Text())
						if line != "" {

							if len(line) > d.width {

								for i := 0; i < len(line); i += d.width {

									end := i + d.width
									if end > len(line) {
										end = len(line)
									}

									d.rows = append(d.rows, []byte(line[i:end]))
									d.rendered = append(d.rendered, []byte(line[i:end]))
								}
							} else {
								d.rows = append(d.rows, []byte(line))
								d.rendered = append(d.rendered, []byte(line))
							}
						}
					}
				}

				d.resetCoordinates()

				d.currentArticleUrl = url
				d.currentSection = ARTICLE_TEXT

				d.SetBottomMessage("HELP: Backspace = go back")
			}
		}
	}
}

func (d *Display) Draw(buf *bytes.Buffer) {

	//log.Default().Printf("len of rendered: %d", len(d.rendered))
	d.endoff = (len(d.rendered) - 1)
	//log.Default().Printf("before: from %d to %d\n", d.startoff, d.endoff)
	if d.endoff >= (d.height - BOTTOM_PADDING) {
		d.endoff = d.height - BOTTOM_PADDING - 1
	}
	if d.endoff+d.startoff <= (len(d.rendered) - 1) {
		d.endoff += d.startoff
	}
	//log.Default().Printf("after: from %d to %d\n", d.startoff, d.endoff)

	//log.Default().Printf("looping from %d to %d\n", d.startoff, d.endoff)
	var printed int
	for i := d.startoff; i <= d.endoff; i++ {

		if i == d.currentRow() && d.currentSection != ARTICLE_TEXT {
			// inverted colors attribute
			buf.WriteString("\x1b[7m")
			// white
			buf.WriteString("\x1b[37m")
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
			// attributes off
			buf.WriteString("\x1b[m")
			// default color
			buf.WriteString("\x1b[39m")
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

	var bottomRightCorner string
	if DebugMode {
		bottomRightCorner = fmt.Sprintf("(y:%v,x:%v) (soff:%v, eoff:%v) (h:%v,w:%v)", d.cy, d.cx, d.startoff, d.endoff, d.height, d.width)
	} else {
		bottomRightCorner = fmt.Sprintf("%d/%d", d.cy+d.startoff, len(d.rendered))
	}
	padding := d.width - utf8.RuneCountInString(d.bottomBarMsg) - 1

	buf.WriteString("\x1b[7m")
	buf.WriteString("\x1b[37m")
	buf.WriteString(fmt.Sprintf("%s %*s\r\n", d.bottomBarMsg, padding, bottomRightCorner))
	buf.WriteString("\x1b[m")
	buf.WriteString("\x1b[39m")
}

/*
openWithBrowser opens the selected article with the default browser for desktop environment.
The environment variable DISPLAY is used to detect if the app is running on a headless machine.
see: https://wiki.debian.org/DefaultWebBrowser
*/
func (d *Display) openWithBrowser(url string) {

	if url != "" {
		err := exec.Command("xdg-open", url).Run()
		if err != nil {
			switch e := err.(type) {
			case *exec.Error:
				d.SetBottomMessage(fmt.Sprintf("failed executing: %v", err))
			case *exec.ExitError:
				d.SetBottomMessage(fmt.Sprintf("command exit with code: %v", e.ExitCode()))
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

func isLynxPresent() bool {
	path, err := exec.LookPath("lynx")
	return err == nil && path != ""
}

/*
openWithLynx opens the selected article with lynx.
see: https://lynx.invisible-island.net/
*/
func (d *Display) openWithLynx(url string) {

	cmd := exec.Command("lynx", url)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Default().Println(err)
	}

	if err := cmd.Wait(); err != nil {
		log.Default().Println(err)
	}
}
