package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/giulianopz/newscanoe/pkg/display"
	"github.com/giulianopz/newscanoe/pkg/termios"
)

var (
	quitC = make(chan bool, 0)
	sigC  = make(chan os.Signal, 1)

	signals = []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGWINCH,
	}
)

func main() {

	flag.BoolVar(&display.DebugMode, "d", false, "enable debug mode")
	flag.Parse()

	origTermios := termios.EnableRawMode(os.Stdin.Fd())
	defer termios.DisableRawMode(os.Stdin.Fd(), origTermios)

	d := display.New(os.Stdin.Fd())

	signal.Notify(sigC, signals...)

	go func() {
		for {
			s := <-sigC
			if s == syscall.SIGWINCH {
				d.SetWindowSize(os.Stdin.Fd())
				d.RefreshScreen()
			} else {
				d.Quit(quitC)
			}
		}
	}()

	go func() {

		defer func() {
			if r := recover(); r != nil {
				d.SetBottomMessage(fmt.Sprintf("recover: %v", r))
			}
		}()

		for {
			d.RefreshScreen()
			d.ProcessKeyStroke(os.Stdin.Fd(), quitC)
		}
	}()

	<-quitC
}
