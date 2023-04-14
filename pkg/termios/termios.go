package termios

import (
	"log"

	"golang.org/x/sys/unix"
)

// TODO use pkg.go.dev/golang.org/x/term for portability

func DisableRawMode(fd uintptr, previousState unix.Termios) {
	log.Default().Println("disabling raw mode")
	if err := unix.IoctlSetTermios(int(fd), unix.TCSETS, &previousState); err != nil {
		log.Panicf("cannot set attr: %v", err)
	}
}

func EnableRawMode(fd uintptr) unix.Termios {
	log.Default().Println("enabling raw mode")
	termios, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	if err != nil {
		log.Panicf("cannot get attr: %v", err)
	}

	var previousState unix.Termios = *termios

	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := unix.IoctlSetTermios(int(fd), unix.TCSETS, termios); err != nil {
		log.Panicf("cannot set attr: %v", err)
	}
	return previousState
}

func GetWindowSize(fd int) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		// TODO impl fallback strategy
		return -1, -1, err
	}
	return int(ws.Col), int(ws.Row), nil
}
