package main

import (
	"fmt"
	"os"

	"github.com/giulianopz/newscanoe/pkg/editor"
	"github.com/giulianopz/newscanoe/pkg/termios"
)

var (
	in = os.Stdin.Fd()
)

func main() {

	origTermios := termios.EnableRawMode(in)
	defer termios.DisableRawMode(in, origTermios)

	e := editor.New()
	//e.SetStatusMessage("HELP: Ctrl-S = save | Ctrl-Q = quit | Ctrl-F = find")
	e.RefreshScreen()
	err := e.GetURLs()
	if err != nil {
		fmt.Fprintf(os.Stdout, err.Error())
	}

	quit := make(chan bool, 0)

	go func() {
		e.ProcessKeyStrokes(in, quit)
	}()

	<-quit
}
