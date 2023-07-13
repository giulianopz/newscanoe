// https://www.xfree86.org/current/ctlseqs.html
package xterm

const (
	DISABLE_MOUSE_TRACKING  = "\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l"
	CLEAR_SCROLLBACK_BUFFER = "\033[3J"
)
