package main

import (
	"flag"
	"io"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
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

	if !display.DebugMode {
		log.SetOutput(io.Discard)
	}

	origTermios := termios.EnableRawMode(os.Stdin.Fd())
	defer termios.DisableRawMode(os.Stdin.Fd(), origTermios)

	d := display.New(os.Stdin.Fd())

	signal.Notify(sigC, signals...)

	go func() {
		for {
			s := <-sigC
			if s == syscall.SIGWINCH {
				if err := d.SetWindowSize(os.Stdin.Fd()); err != nil {
					log.Default().Printf("cannot reset window size: %v\n", err)
				}
				d.RefreshScreen()
			} else {
				d.Quit(quitC)
			}
		}
	}()

	go func() {

		defer func() {
			if r := recover(); r != nil {
				log.Default().Printf("recover from: %v\nstack trace: %v\n", r, string(debug.Stack()))
			}
		}()

		for !d.Quitting {
			d.RefreshScreen()
			d.ProcessKeyStroke(os.Stdin.Fd(), quitC)
		}
	}()

	<-quitC
}
