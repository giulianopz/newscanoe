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

	rows     []string
	rendered [][]byte

	pages          int
	currentPage    int
	currentSection string
	prev           string
	next           string
}

func New() *Display {
	return &Display{
		cx: 1,
		cy: 1,
	}
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

func (d *Display) MoveCursor(dir byte) {
	switch dir {
	case ARROW_LEFT:
		d.cx--
	case ARROW_RIGHT:
		d.cx++
	case ARROW_DOWN:
		d.cy++
	case ARROW_UP:
		d.cy--
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

					//fmt.Fprintf(os.Stdout, "debug: seq=%v\r\n", seq)

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

func (d *Display) ProcessKeyStroke(fd uintptr, quit chan bool) {

	for {

		input := readKeyStroke(fd)

		switch input {
		case ctrlPlus('q'), 'q':
			{
				fmt.Fprint(os.Stdout, "\x1b[2J")
				fmt.Fprint(os.Stdout, "\x1b[H")
				quit <- true
			}
		case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
			d.MoveCursor(input)
		default:
			//fmt.Fprintf(os.Stdout, "keystroke: %v\r\n", input)
		}
		fmt.Fprintf(os.Stdout, "\x1b[%d;%dH", d.cy, d.cx)
	}
}

func (d *Display) RefreshScreen() {

	buf := &bytes.Buffer{}

	// hide cursor
	buf.WriteString("\x1b[?25l")
	// move cursor to (y,x)
	buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", d.cy, d.cx))
	// show cursor
	buf.WriteString("\x1b[?25h")
	// erase everything after cursor
	buf.WriteString("\x1b[0J")

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

func (d *Display) GetURLs() error {

	home := os.Getenv("HOME")

	file, _ := os.Open(home + "/.newsboat/urls")
	defer file.Close()

	d.rows = make([]string, 0)
	d.rendered = make([][]byte, 0)

	fscanner := bufio.NewScanner(file)
	for fscanner.Scan() {

		url := fscanner.Bytes()
		if !strings.Contains(string(url), "#") {
			d.rows = append(d.rows, string(url))
			d.rendered = append(d.rendered, url)
		}
	}
	return nil
}

func (d *Display) Draw() {

	buf := &bytes.Buffer{}

	var printed int
	for i := 0; i < len(d.rows); i++ {
		if i < d.height-3 {
			for j := 0; j < len(d.rows[i]); j++ {
				if j < (d.width) {
					buf.WriteString(string(d.rendered[i][j]))
				}
			}
			buf.WriteString("\r\n")
			printed++
		}
	}

	for ; printed < d.height-2; printed++ {
	}

	for k := 0; k < d.width; k++ {
		buf.WriteString("-")
	}
	buf.WriteString("\r\n")

	buf.WriteString(fmt.Sprintf("%s\r\n", d.msgtatus))

	// move cursor to (y,x)
	buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", d.cy, d.cx))
	// show cursor
	buf.WriteString("\x1b[?25h")

	fmt.Fprint(os.Stdout, buf)
}
