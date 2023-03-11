package main

import (
	"fmt"
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

	d := display.New()
	d.SetStatusMessage("HELP: Ctrl-Q = quit")
	d.RefreshScreen()

	d.SetWindowSize(in)

	err := d.GetURLs()
	if err != nil {
		fmt.Fprintf(os.Stdout, err.Error())
	}
	d.Draw()

	quit := make(chan bool, 0)

	go func() {
		d.ProcessKeyStrokes(in, quit)
	}()

	<-quit
}
