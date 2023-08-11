package xterm

const (
	// https://www.xfree86.org/current/ctlseqs.html#Mouse%20Tracking
	DISABLE_MOUSE_TRACKING  = "\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l"
	CLEAR_SCROLLBACK_BUFFER = 3
	// https://www.xfree86.org/current/ctlseqs.html#Bracketed%20Paste%20Mode
	ENABLE_BRACKETED_PASTE  = "\x1b[?2004h"
	DISABLE_BRACKETED_PASTE = "\x1b[?2004l"
)
