package display

import (
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/internal/ascii"
	"github.com/giulianopz/newscanoe/internal/util"
	"golang.org/x/sys/unix"
)

// keys
const (
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
)

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

				if seq[1] == '2' && seq[2] == '0' {
					subseq := make([]byte, 2)

					log.Default().Printf("4rd keystroke byte: %v", subseq[0])
					log.Default().Printf("6th keystroke byte: %v", subseq[1])

					if _, err := unix.Read(int(fd), subseq); err != nil {
						return QUIT
					}
					if subseq[0] == '0' && subseq[1] == '~' {
						log.Default().Println("started pasting")
						return ascii.NULL
					} else if subseq[0] == '1' && subseq[1] == '~' {
						log.Default().Println("done pasting")
						return ascii.NULL
					}
				}

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
					case 'B': // 66
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
	if d.editingMode {
		d.whileEditing(input)
	} else {
		d.whileReading(input, quitC)
	}
}

func (d *display) whileReading(input byte, quitC chan bool) {
	switch input {

	case ctrlPlus('q'), 'q':
		d.Quit(quitC)

	case 'r':
		if d.currentSection == URLS_LIST {
			d.fetchFeed(string(d.raw[d.currentRow()]))
		}

	case 'a':
		if d.currentSection == URLS_LIST {
			d.enterEditingMode()
		}

	case 'R':
		if d.currentSection == URLS_LIST {
			d.fetchAllFeeds()
		}

	case 'o':
		if d.currentSection == ARTICLES_LIST {
			if !util.IsHeadless() {
				if err := util.OpenWithBrowser(string(d.raw[d.currentRow()])); err != nil {
					d.setTmpBottomMessage(1*time.Second, err.Error())
				}
			}
		}

	case 'l':
		if d.currentSection == ARTICLES_LIST {
			if util.IsLynxPresent() {
				if err := util.OpenWithLynx(string(d.raw[d.currentRow()])); err != nil {
					d.setTmpBottomMessage(1*time.Second, err.Error())
				}
			}
		}

	case ARROW_UP, ARROW_DOWN:
		d.moveCursor(input)

	case PAGE_UP, PAGE_DOWN:
		d.scroll(input)

	case ascii.ENTER:
		{

			switch d.currentSection {
			case URLS_LIST:
				{
					d.trackPos()
					if err := d.loadArticleList(d.currentUrl()); err != nil {
						d.restorePos()
					} else {
						d.resetCurrentPos()
					}
				}
			case ARTICLES_LIST:
				{
					d.trackPos()
					if err := d.loadArticleText(d.currentUrl()); err != nil {
						d.restorePos()
					} else {
						d.resetCurrentPos()
					}
				}
			}
		}

	case ascii.BACKSPACE:
		{
			switch d.currentSection {
			case ARTICLES_LIST:
				{
					if err := d.LoadFeedList(); err != nil {
						log.Default().Printf("cannot load urls: %v", err)
					}
					d.currentFeedUrl = ""
					d.restorePos()
				}
			case ARTICLE_TEXT:
				{
					if err := d.loadArticleList(d.currentFeedUrl); err != nil {
						log.Default().Printf("cannot load article of feed with url %q: %v", d.currentFeedUrl, err)
					}
					d.currentArticleUrl = ""
					d.restorePos()
				}
			}
		}

	default:
		{
			log.Default().Printf("unhandled: %v\n", input)
		}
	}
}

func (d *display) whileEditing(input byte) {
	switch {

	case input == ascii.NULL:
		log.Default().Println("pasting...")

	case input == ARROW_LEFT:
		if d.current.cx > 1 {
			d.current.cx--
		}

	case input == ARROW_RIGHT:
		if d.current.cx < len(d.editingBuf)+1 {
			d.current.cx++
		}

	case input == DEL_KEY:
		{
			i := d.current.cx - 1
			if i < len(d.editingBuf) {
				d.deleteCharAt(i)
				d.setBottomMessage(strings.Join(d.editingBuf, ""))
			}
		}

	case input == ascii.BACKSPACE:
		{
			if len(d.editingBuf) != 0 && d.current.cx == len(d.editingBuf)+1 {
				d.deleteCharAt(len(d.editingBuf) - 1)
				d.setBottomMessage(strings.Join(d.editingBuf, ""))
				d.current.cx--
			}
		}
	case input == ascii.ENTER:
		{
			d.addEnteredFeedUrl()
		}
	case util.IsLetter(input), util.IsDigit(input), util.IsSpecialChar(input):
		{
			if d.current.cx < (d.width - utf8.RuneCountInString(d.bottomRightCorner) - 1) {
				i := d.current.cx - 1
				d.insertCharAt(string(input), i)
				d.setBottomMessage(strings.Join(d.editingBuf, ""))
				d.current.cx++
			}
		}
	case input == QUIT:
		{
			d.setBottomMessage(urlsListSectionMsg)
			d.setTmpBottomMessage(1*time.Second, "editing aborted!")
			d.exitEditingMode()
			d.resetCurrentPos()
		}
	default:
		{
			log.Default().Printf("unhandled: %v\n", input)
		}
	}
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

func (d *display) moveCursor(direction byte) {

	var lastRow int = len(d.rendered) - 1

	switch direction {
	case ARROW_DOWN:

		if d.currentSection == ARTICLE_TEXT {
			if d.current.endoff < lastRow {
				d.current.startoff++
			} else {
				d.current.cy = d.getContentWindowLen()
			}
			return
		}

		if d.current.cy < d.getContentWindowLen() {
			if d.currentRow()+1 <= lastRow {
				d.current.cy++
			}
		} else if d.current.endoff < lastRow {
			d.current.startoff++
		}
	case ARROW_UP:

		if d.currentSection == ARTICLE_TEXT {
			if d.current.startoff > 0 {
				d.current.startoff--
			}
			return
		}

		if d.current.cy > 1 {
			if d.currentRow()-1 <= lastRow {
				d.current.cy--
			}
		} else if d.current.startoff > 0 {
			d.current.startoff--
		}
	}
}

func (d *display) scroll(dir byte) {

	switch dir {
	case PAGE_DOWN:
		{

			var lastRow int = len(d.rendered) - 1
			if d.current.endoff == lastRow {
				return
			}

			d.current.cy = d.getContentWindowLen()

			firstItemInNextPage := d.current.endoff + 1
			if firstItemInNextPage < lastRow {
				d.current.startoff = firstItemInNextPage
			} else {
				d.current.startoff++
				d.current.endoff = lastRow
			}
		}
	case PAGE_UP:
		{
			d.current.cy = 1

			if d.current.startoff == 0 {
				return
			}

			firstItemInPreviousPage := d.current.startoff - d.getContentWindowLen()
			if firstItemInPreviousPage >= 0 {
				d.current.startoff = firstItemInPreviousPage
			} else {
				d.current.startoff = 0
				d.current.endoff = d.getContentWindowLen() - 1
			}
		}
	}
}
