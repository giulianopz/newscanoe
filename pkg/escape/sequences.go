package escape

import (
	"fmt"

	"github.com/giulianopz/newscanoe/pkg/ascii"
)

/*
see: https://vt100.net/docs/vt100-ug/chapter3.html
*/

const (

	// control sequence introducer
	csi = string(ascii.ESC) + "["

	MOVE_CURSOR_FMT = csi + "%d;%dH"
	SGR_FMT         = csi + "%dm"

	ERASE_ENTIRE_SCREEN = csi + "2J"
	HIDE_CURSOR         = csi + "?25l"
	SHOW_CURSOR         = csi + "?25h"
	ATTRIBUTES_OFF      = csi + "m"
	REVERSE_COLOR       = csi + "7m"
	DEFAULT_FG_COLOR    = csi + "39m"
)

func MoveCursor(y, x int) string {
	return fmt.Sprintf(MOVE_CURSOR_FMT, y, x)
}

func SelectGraphicRendition(n int) string {
	return fmt.Sprintf(SGR_FMT, n)
}
