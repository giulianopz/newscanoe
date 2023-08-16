package newscanoe

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/giulianopz/newscanoe/internal/display"
	"github.com/giulianopz/newscanoe/internal/termios"
	"github.com/giulianopz/newscanoe/internal/xterm"
)

var (
	quitC = make(chan bool)
	sigC  = make(chan os.Signal, 1)

	signals = []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGWINCH,
	}
)

func Run(debugMode bool) {

	signal.Notify(sigC, signals...)

	origTermios := termios.EnableRawMode(os.Stdin.Fd())
	defer termios.DisableRawMode(os.Stdin.Fd(), origTermios)

	d := display.New(debugMode)
	w, h, err := termios.GetWindowSize(int(os.Stdin.Fd()))
	if err != nil {
		log.Panicln(err)
	}
	d.SetWindowSize(w, h)

	if err := d.LoadConfig(); err != nil {
		log.Panicln(err)
	}

	if err := d.LoadCache(); err != nil {
		log.Panicln(err)
	}

	if err := d.LoadFeedList(); err != nil {
		log.Panicln(err)
	}

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

		fmt.Fprintf(os.Stdout, xterm.DISABLE_MOUSE_TRACKING)
		fmt.Fprintf(os.Stdout, xterm.ENABLE_BRACKETED_PASTE)

		for d.ListenToKeyStrokes {
			d.RefreshScreen()
			d.ProcessKeyStroke(os.Stdin.Fd(), quitC)
		}
	}()

	<-quitC
}
