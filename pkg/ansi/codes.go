package ansi

/*
see: https://en.wikipedia.org/wiki/ANSI_escape_code
*/

const (
	/* colors */
	RED   = 31
	GREEN = 32
	WHITE = 37
	// display attributes
	ALL_ATTRIBUTES_OFF = 0
	REVERSE_COLOR      = 7
	BOLD               = 1
	DEFAULT_FG_COLOR   = 39
	// erase
	ERASE_ENTIRE_SCREEN = 2
	//
	CURSOR = 25
)
