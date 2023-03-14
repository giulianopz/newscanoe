package main

import (
	"os"
	"os/signal"
	"syscall"

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

	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, syscall.SIGWINCH)

	go func() {
		for {
			<-sigC
			d.SetWindowSize(in)
			d.RefreshScreen()
		}
	}()

	quit := make(chan bool, 0)

	go func() {
		for {
			d.RefreshScreen()
			d.ProcessKeyStroke(in, quit)
		}
	}()

	<-quit
}

var (
	terminationSignals = []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT,
	}
)
