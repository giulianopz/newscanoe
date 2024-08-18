package newscanoe

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/giulianopz/newscanoe/internal/display"
	"github.com/giulianopz/newscanoe/internal/termios"
	"github.com/giulianopz/newscanoe/internal/xterm"
)

var (
	sigC = make(chan os.Signal, 1)

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
			switch <-sigC {
			case syscall.SIGWINCH:
				{
					w, h, err := termios.GetWindowSize(int(os.Stdin.Fd()))
					if err != nil {
						log.Default().Printf("cannot reset window size: %v\n", err)
					} else {
						d.SetWindowSize(w, h)
						d.RefreshScreen()
					}
				}
			default:
				d.QuitC <- true
			}
		}
	}()

	fmt.Fprintf(os.Stdout, xterm.DISABLE_MOUSE_TRACKING)
	fmt.Fprintf(os.Stdout, xterm.ENABLE_BRACKETED_PASTE)

	for {
		select {

		default:

			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Default().Printf("recover from: %v\nstack trace: %v\n", r, string(debug.Stack()))
						d.SetTmpBottomMessage(2*time.Second, "something bad happened: check the logs")
					}
				}()

				d.RefreshScreen()
				input := d.ReadKeyStroke(os.Stdin.Fd())
				d.ProcessKeyStroke(input)
			}()

		case <-d.QuitC:
			d.Clear()
			return
		}
	}
}
