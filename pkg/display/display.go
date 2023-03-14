package display

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/giulianopz/newscanoe/pkg/termios"
	"golang.org/x/sys/unix"
)

var bottomPadding int = 3

const (
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
)

type Display struct {
	// cursor's position within the file
	cx int
	cy int
	// file's position
	rowoff int
	coloff int
	// win size
	height int
	width  int

	msgtatus string
	infotime time.Time

	origTermios unix.Termios

	rows     [][]byte
	rendered [][]byte

	pages          int
	currentPage    int
	currentSection string
	prev           string
	next           string
}

func New(in uintptr) *Display {
	d := &Display{
		cx: 1,
		cy: 1,
	}

	if err := d.LoadURLs(); err != nil {
		log.Fatal(err)
	}

	d.SetStatusMessage("HELP: Ctrl-Q = quit | Ctrl-r = reload | Ctrl-R = reload all")
	d.SetWindowSize(in)

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
			d.cx = 0
		}
	case ARROW_DOWN:
		if d.cy < (d.height - bottomPadding) {
			if (d.cx - 1) <= (len(d.rendered[d.cy]) - 1) {
				d.cy++
			}
		}
	case ARROW_UP:
		if d.cy > 1 {
			if (d.cx - 1) <= (len(d.rendered[d.cy-2]) - 1) {
				d.SetStatusMessage(fmt.Sprintf("%d <= %d", d.cx-1, (len(d.rendered[d.cy-1]) - 1)))
				d.cy--
			}
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
	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		d.MoveCursor(input)
	default:
		//fmt.Fprintf(os.Stdout, "keystroke: %v\r\n", input)
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

	home := os.Getenv("HOME")

	file, _ := os.Open(home + "/.newsboat/urls")
	defer file.Close()

	d.rows = make([][]byte, 0)
	d.rendered = make([][]byte, 0)

	fscanner := bufio.NewScanner(file)
	for fscanner.Scan() {

		url := fscanner.Bytes()
		if !strings.Contains(string(url), "#") {
			d.rows = append(d.rows, url)
			d.rendered = append(d.rendered, url)
		}
	}
	return nil
}

func (d *Display) Draw(buf *bytes.Buffer) {

	var printed int
	for i := 0; i < len(d.rows) && i < (d.height-bottomPadding); i++ {

		for j := 0; j < len(d.rows[i]); j++ {
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

	tracking := fmt.Sprintf("(y:%v,x:%v)(h:%v,w:%v)", d.cy, d.cx, d.height, d.width)
	buf.WriteString(fmt.Sprintf("%s %135s\r\n", d.msgtatus, tracking))
}
