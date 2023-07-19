package display

import (
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/pkg/ansi"
	"github.com/giulianopz/newscanoe/pkg/ascii"
	"github.com/giulianopz/newscanoe/pkg/util"
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
			d.loadFeed(string(d.raw[d.currentRow()]))
		}

	case 'a':
		if d.currentSection == URLS_LIST {
			d.enterEditingMode()
		}

	case 'R':
		if d.currentSection == URLS_LIST {
			d.loadAllFeeds()
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
				d.loadArticlesList(d.currentUrl())
			case ARTICLES_LIST:
				d.loadArticleText(d.currentUrl())
			}
		}

	case ascii.BACKSPACE:
		{
			switch d.currentSection {
			case ARTICLES_LIST:
				if err := d.LoadURLs(); err != nil {
					log.Default().Printf("cannot load urls: %v", err)
				}
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

func (d *display) whileEditing(input byte) {
	switch {

	case input == ascii.NULL:
		log.Default().Println("pasting...")

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
			d.exitEditingMode(ansi.WHITE)
			d.resetCoordinates()
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

func (d *display) moveCursor(dir byte) {

	var renderedRowsLen int = len(d.rendered) - 1

	switch dir {
	case ARROW_DOWN:
		if d.cy < (d.height - BOTTOM_PADDING - TOP_PADDING) {
			if d.currentRow()+1 <= renderedRowsLen {
				d.cy++
			}
		} else if d.endoff < renderedRowsLen {
			d.startoff++
		}
	case ARROW_UP:
		if d.cy > 1 {
			if d.currentRow()-1 <= renderedRowsLen {
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
			var renderedRowsLen int = len(d.rendered) - 1

			if d.endoff == renderedRowsLen {
				d.cy = d.height - BOTTOM_PADDING - TOP_PADDING
				return
			}

			firstItemInNextPage := d.endoff + 1
			if firstItemInNextPage < renderedRowsLen {
				d.startoff = firstItemInNextPage
			} else {
				d.startoff++
				d.endoff = renderedRowsLen
			}

			d.cy = d.height - BOTTOM_PADDING - TOP_PADDING
		}
	case PAGE_UP:
		{
			if d.startoff == 0 {
				d.cy = 1
				return
			}

			firstItemInPreviousPage := d.startoff - (d.height - BOTTOM_PADDING - TOP_PADDING)
			if firstItemInPreviousPage >= 0 {
				d.startoff = firstItemInPreviousPage
			} else {
				d.startoff = 0
				d.endoff = d.height - BOTTOM_PADDING - TOP_PADDING - 1
			}

			d.cy = 1
		}
	}
}
