package main

import (
	"os"

	"github.com/giulianopz/newscanoe/pkg/display"
	"github.com/giulianopz/newscanoe/pkg/termios"
)

var (
	in = os.Stdin.Fd()
)

func main() {

	origTermios := termios.EnableRawMode(in)
	defer termios.DisableRawMode(in, origTermios)

	d := display.New(in)

	quit := make(chan bool, 0)

	go func() {
		for {
			d.RefreshScreen()
			d.ProcessKeyStroke(in, quit)
		}
	}()

	<-quit
}
