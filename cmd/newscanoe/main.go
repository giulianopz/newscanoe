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

	quitC = make(chan bool, 0)
	sigC  = make(chan os.Signal, 1)

	signals = []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGWINCH,
	}
)

func main() {

	origTermios := termios.EnableRawMode(in)
	defer termios.DisableRawMode(in, origTermios)

	d := display.New(in)

	signal.Notify(sigC, signals...)

	go func() {
		for {
			s := <-sigC
			if s == syscall.SIGWINCH {
				d.SetWindowSize(in)
				d.RefreshScreen()
			} else {
				d.Quit(quitC)
			}
		}
	}()

	go func() {
		for {
			d.RefreshScreen()
			d.ProcessKeyStroke(in, quitC)
		}
	}()

	<-quitC
}
