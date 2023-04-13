package termios

import (
	"log"

	"golang.org/x/sys/unix"
)

// TODO use pkg.go.dev/golang.org/x/term for portability

const (
	TCIFLUSH  = 0
	TCOFLUSH  = 1
	TCIOFLUSH = 2

	TCSANOW   = 0
	TCSADRAIN = 1
	TCSAFLUSH = 2
)

func tcgetattr(fd uintptr) (*unix.Termios, error) {
	return unix.IoctlGetTermios(int(fd), unix.TCGETS)
}

func tcsetattr(fd, action uintptr, t *unix.Termios) error {
	var request uintptr
	switch action {
	case TCSANOW:
		request = unix.TCSETS
	case TCSADRAIN:
		request = unix.TCSETSW
	case TCSAFLUSH:
		request = unix.TCSETS
	default:
		return unix.EINVAL
	}
	return unix.IoctlSetTermios(int(fd), uint(request), t)
}

func DisableRawMode(fd uintptr, orig_termios unix.Termios) {
	if err := tcsetattr(fd, TCSAFLUSH, &orig_termios); err != nil {
		log.Panicf("cannot set attr: %v", err)
	}
}

func EnableRawMode(fd uintptr) unix.Termios {
	termios, err := tcgetattr(fd)
	if err != nil {
		log.Panicf("cannot get attr: %v", err)
	}

	var origTermios unix.Termios = *termios

	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0
	if err := tcsetattr(fd, TCSAFLUSH, termios); err != nil {
		log.Panicf("cannot set attr: %v", err)
	}
	return origTermios
}

func GetWindowSize(fd int) (width, height int, err error) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		// TODO impl fallback strategy
		return -1, -1, err
	}
	return int(ws.Col), int(ws.Row), nil
}
