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

	rows [][]byte

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

func (d *Display) ProcessKeyStrokes(fd uintptr, quit chan bool) {

	for {
		buf := make([]byte, 1)
		_, err := unix.Read(int(fd), buf)
		if err != nil {
			log.Fatal(err)
		}

		input := buf[0]

		switch input {
		case ctrlPlus('q'), 'q':
			{
				quit <- true
			}
		default:
			fmt.Fprintf(os.Stdout, "keystroke: %v\r\n", input)
		}
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

	d.rows = make([][]byte, 0)
	fscanner := bufio.NewScanner(file)
	for fscanner.Scan() {

		url := fscanner.Bytes()
		if !strings.Contains(string(url), "#") {
			d.rows = append(d.rows, url)
		}
	}
	return nil
}

func (d *Display) Draw() {

	var printed int
	for i := 0; i < len(d.rows); i++ {
		if i < d.height-2 {
			for j := 0; j < len(d.rows[i]); j++ {
				if j < (d.width) {
					fmt.Fprint(os.Stdout, string(d.rows[i][j]))
				}
			}
			fmt.Fprint(os.Stdout, "\r\n")
			printed++
		}
	}

	for ; printed < d.height-2; printed++ {
	}

	for k := 0; k < d.width; k++ {
		fmt.Fprint(os.Stdout, "-")
	}
	fmt.Fprint(os.Stdout, "\r\n")

	fmt.Fprintf(os.Stdout, "%s\r\n", d.msgtatus)
}
