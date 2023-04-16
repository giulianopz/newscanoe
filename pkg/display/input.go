package display

import (
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/pkg/ascii"
	"github.com/giulianopz/newscanoe/pkg/escape"
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

	// TODO copy from clipboard

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
				d.LoadURLs()
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
