package ansi

/*
see: https://en.wikipedia.org/wiki/ANSI_escape_code
*/

const (
	/* colors */
	BLACK_FG = 30
	RED_FG   = 31
	GREEN_FG = 32
	WHITE_FG = 37

	BLACK_BG = 40

	// display attributes
	ALL_ATTRIBUTES_OFF = 0
	BOLD               = 1
	FAINT              = 2
	REVERSE_COLOR      = 7
	SET_FG_COLOR       = 38
	SET_BG_COLOR       = 48
	// erase
	ERASE_ENTIRE_SCREEN = 2
	//
	CURSOR = 25
)
