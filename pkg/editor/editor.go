package editor

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type Editor struct {
	// cursor's position within the file
	cx int
	cy int
	// file's position
	rowoff int
	coloff int

	msgtatus string
	infotime time.Time

	origTermios unix.Termios

	urls []string
}

func New() *Editor {
	return &Editor{
		cx: 1,
		cy: 1,
	}
}

func ctrlPlus(k byte) byte {
	return k & 0x1f
}

func (e *Editor) ProcessKeyStrokes(fd uintptr, quit chan bool) {

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

func (e *Editor) RefreshScreen() {

	buf := &bytes.Buffer{}

	// hide cursor
	buf.WriteString("\x1b[?25l")
	// move cursor to (y,x)
	buf.WriteString(fmt.Sprintf("\x1b[%d;%dH", e.cy, e.cx))
	// show cursor
	buf.WriteString("\x1b[?25h")
	// erase everything after cursor
	buf.WriteString("\x1b[0J")

	fmt.Fprint(os.Stdout, buf)
}

func (e *Editor) SetStatusMessage(msg string) {
	e.msgtatus = msg
}

func (e *Editor) GetURLs() error {

	home := os.Getenv("HOME")

	file, _ := os.Open(home + "/.newsboat/urls")
	defer file.Close()

	e.urls = make([]string, 0)
	fscanner := bufio.NewScanner(file)
	for fscanner.Scan() {

		url := fscanner.Text()
		if !strings.Contains(url, "#") {
			e.urls = append(e.urls, url)
		}
	}

	return nil
}
