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
	"golang.org/x/exp/slices"
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

const (
	urlsListSectionMsg     = "HELP: q = quit | r = reload | R = reload all | a = add a feed"
	articlesListSectionMsg = "HELP: Enter = view article | Backspace = go back"
	articleTextSectionMsg  = "HELP: Backspace = go back"
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
	// message displayed in the bottom bar
	bottomBarMsg string
	// previous state of the terminal
	origTermios unix.Termios
	// dislay raw text
	raw [][]byte
	// dislay rendered text
	rendered [][]byte
	// gob cache
	cache *cache.Cache

	liveEditing       bool
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

func (d *Display) Quit(quitC chan bool) {
	fmt.Fprint(os.Stdout, "\x1b[?25h")
	fmt.Fprint(os.Stdout, "\x1b[2J")
	fmt.Fprint(os.Stdout, "\x1b[H")
	quitC <- true
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
	d.raw = make([][]byte, 0)
	d.rendered = make([][]byte, 0)
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

// TODO move only line-by-line or scroll page-by-page
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
		} else if d.cy >= 1 && d.cy < (d.height-BOTTOM_PADDING) && d.cy+1 < len(d.rendered) {
			d.cy++
			d.cx = 1
		}
	case ARROW_DOWN:
		log.Default().Println("down")
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

func readKeyStroke(fd uintptr) byte {

	input := make([]byte, 1)
	for {

		_, err := unix.Read(int(fd), input)
		if err != nil {
			//TODO use new slog
			log.Default().Println(err)
		}
		log.Default().Printf("1st keystroke byte: %v", input)

		if input[0] == '\x1b' {

			seq := make([]byte, 3)

			_, err := unix.Read(int(fd), seq)
			if err != nil {
				return QUIT
			}

			log.Default().Printf("2nd keystroke byte: %v", seq[0])

			if seq[0] == '[' {

				log.Default().Printf("3rd keystroke byte: %v", seq[1])
				log.Default().Printf("4th keystroke byte: %v", seq[2])

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

func isLetter(input byte) bool {
	return (input >= 'A' && input <= 'Z') || (input >= 'a' && input <= 'z')
}

func isDigit(input byte) bool {
	return input >= '0' && input <= '9'
}

var specialChars = []byte{
	';', ',', '/', '?', ':', '@', '&', '=', '+', '$', '-', '_', '.', '!', '~', '*', '(', ')', '#', 96,
}

/*
isSpecialChar returns if a given input char is a reserved char as per RFC 3986
see paragraph 2.2: https://www.ietf.org/rfc/rfc3986.txt
*/
func isSpecialChar(input byte) bool {
	return slices.Contains(specialChars, input)
}

func (d *Display) ProcessKeyStroke(fd uintptr, quitC chan bool) {

	input := readKeyStroke(fd)

	// TODO copy from clipboard

	if d.liveEditing {
		switch {

		//TODO move cursor along bottom bar

		/* 		case input == ARROW_LEFT:
		   			if d.cx > 1 {
		   				d.cx--
		   			}
		   		case input == ARROW_RIGHT:
		   			if d.cx < d.width {
		   				d.cx++
		   			} */
		case input == BACKSPACE:
			{
				d.cx--
				d.SetBottomMessage(d.bottomBarMsg[:len(d.bottomBarMsg)-1])
			}
		case input == ENTER:
			{
				url := strings.TrimSpace(d.bottomBarMsg)

				// TODO extract this block in a func since repeated
				fp := gofeed.NewParser()
				parsedFeed, err := fp.ParseURL(url)
				if err != nil {
					log.Default().Println(err)
					d.SetTmpBottomMessage(3*time.Second, "cannot parse feed!")
					return
				}

				// TODO extract this block in a func since too long
				present, err := util.IsUrlAlreadyPresent(url)
				if err != nil {
					log.Default().Println(err)
					d.SetTmpBottomMessage(3*time.Second, "cannot check if url is already present in config file!")
					return
				}

				if present {
					d.SetTmpBottomMessage(3*time.Second, "url already present!")
					return
				}

				if err := util.AppendUrl(url); err != nil {
					log.Default().Println(err)
					d.SetTmpBottomMessage(3*time.Second, "cannot save url in config file!")
					return
				}

				d.raw = append(d.raw, []byte(d.bottomBarMsg))
				d.rendered = append(d.rendered, []byte(parsedFeed.Title))

				d.SetBottomMessage(urlsListSectionMsg)
				d.SetTmpBottomMessage(3*time.Second, "saved: type r to reload!")
				d.liveEditing = false
				d.cx = 1
				d.cy = len(d.rendered)
			}
		case isLetter(input), isDigit(input), isSpecialChar(input):
			{
				if d.cx < d.width {
					d.cx++
					d.SetBottomMessage(fmt.Sprintf("%s%s", d.bottomBarMsg, string(input)))
				}
			}
		case input == 11:
			{
				d.SetBottomMessage(urlsListSectionMsg)
				d.SetTmpBottomMessage(1*time.Second, "editing aborted!")
				d.liveEditing = false
				d.cx = 1
				d.cy = 1
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
			d.LoadFeed(string(d.raw[d.currentRow()]))
		}

	case ctrlPlus('a'), 'a':
		if d.currentSection == URLS_LIST {
			log.Default().Println("live editing enabled")
			d.liveEditing = true
			d.cy = d.height - 1
			d.cx = 1
			d.SetBottomMessage("")
		}

	case ctrlPlus('R'), 'R':
		if d.currentSection == URLS_LIST {
			d.LoadAllFeeds()
		}

	case ctrlPlus('o'), 'o':
		if d.currentSection == ARTICLES_LIST {
			if !headless() {
				d.openWithBrowser(string(d.raw[d.currentRow()]))
			}
		}

	case ctrlPlus('l'), 'l':
		if d.currentSection == ARTICLES_LIST {
			if isLynxPresent() {
				d.openWithLynx(string(d.raw[d.currentRow()]))
			}
		}

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.MoveCursor(input)

	case ENTER:
		{
			switch d.currentSection {
			case URLS_LIST:
				d.LoadArticlesList(string(d.raw[d.currentRow()]))
			case ARTICLES_LIST:
				d.LoadArticleText(string(d.raw[d.currentRow()]))
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

	default:
		{
			log.Default().Printf("unhandled: %v\n", input)
		}
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

	empty, err := util.IsEmpty(file)
	if err != nil {
		log.Panicln(err)
	}

	if empty && len(d.cache.GetFeeds()) == 0 {
		d.SetBottomMessage("no feed url: type 'a' to add one now")
	} else {
		cached := make(map[string]bool, 0)

		// TODO show only feeds in config file
		for _, cachedFeed := range d.cache.GetFeeds() {
			cached[cachedFeed.Url] = true
			d.raw = append(d.raw, []byte(cachedFeed.Url))
			d.rendered = append(d.rendered, []byte(cachedFeed.Title))
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {

			url := scanner.Bytes()
			if !strings.Contains(string(url), "#") {

				if _, present := cached[string(url)]; !present {

					d.raw = append(d.raw, url)
					d.rendered = append(d.rendered, url)
					d.cache.AddFeedUrl(string(url))
				}
			}
		}
		d.SetBottomMessage(urlsListSectionMsg)
	}

	d.cy = 1
	d.cx = 1
	d.currentSection = URLS_LIST
	return nil
}

func (d *Display) LoadFeed(url string) {

	fp := gofeed.NewParser()
	parsedFeed, err := fp.ParseURL(url)
	if err != nil {
		log.Default().Println(err)
		d.SetTmpBottomMessage(3*time.Second, "cannot parse feed!")
		return
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

	for id := range d.raw {

		url := string(d.raw[id])

		log.Default().Printf("loading feed #%d from url %s\n", id, url)

		fp := gofeed.NewParser()
		parsedFeed, err := fp.ParseURL(url)
		if err != nil {
			log.Default().Println(err)
			d.SetTmpBottomMessage(3*time.Second, "cannot parse feed!")
			return
		}

		title := strings.TrimSpace(parsedFeed.Title)

		if err := d.cache.AddFeed(parsedFeed, url); err != nil {
			log.Default().Println(err)
			d.SetTmpBottomMessage(3*time.Second, fmt.Sprintf("cannot load feed from url: %s", url))
			return
		}

		d.rendered[id] = []byte(title)

		d.SetBottomMessage(fmt.Sprintf("loading all feeds, please wait........%d/%d", id+1, len(d.raw)))
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
				d.raw = append(d.raw, []byte(item.Url))
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

			d.SetBottomMessage(fmt.Sprintf("%s %s %s", articlesListSectionMsg, browserHelp, lynxHelp))
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

									d.raw = append(d.raw, []byte(line[i:end]))
									d.rendered = append(d.rendered, []byte(line[i:end]))
								}
							} else {
								d.raw = append(d.raw, []byte(line))
								d.rendered = append(d.rendered, []byte(line))
							}
						}
					}
				}

				d.resetCoordinates()

				d.currentArticleUrl = url
				d.currentSection = ARTICLE_TEXT

				d.SetBottomMessage(articleTextSectionMsg)
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
