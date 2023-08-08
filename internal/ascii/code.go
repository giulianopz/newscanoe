package ascii

/*
// ASCII control codes
see:
- https://theasciicode.com.ar/
- https://en.wikipedia.org/wiki/C0_and_C1_control_codes
*/

const (
	NULL      = 00
	ENTER     = 13
	BACKSPACE = 127
	// ANSI escapes always start with ESC which can be represented as \x1b (hexadecimal), or \033 (octal), or 27 (decimal) or \e (special character reference)
	ESC = '\x1b'
)
