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

	d := display.New()
	w, h, err := termios.GetWindowSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Panicln(err)
	}
	d.SetWindowSize(w, h)

	if err := d.LoadCache(); err != nil {
		log.Panicln(err)
	}

	if err := d.LoadURLs(); err != nil {
		log.Panicln(err)
	}
	signal.Notify(sigC, signals...)

	go func() {
		for {
			s := <-sigC
			if s == syscall.SIGWINCH {
				w, h, err := termios.GetWindowSize(int(os.Stdin.Fd()))
				if err != nil {
					log.Default().Printf("cannot reset window size: %v\n", err)
				}
				d.SetWindowSize(w, h)

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
