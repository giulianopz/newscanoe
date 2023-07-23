package ansi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/giulianopz/newscanoe/pkg/ascii"
)

/*
see: https://vt100.net/docs/vt100-ug/chapter3.html
*/

const (

	// control sequence introducer
	csi = string(ascii.ESC) + "["

	MOVE_CURSOR_FMT = csi + "%d;%dH"
	SGR_FMT         = csi + "%sm"
	ERASE_FMT       = csi + "%dJ"
	SET_MODE_FMT    = csi + "%sh"
	RESET_MODE      = csi + "%sl"
)

func MoveCursor(y, x int) string {
	return fmt.Sprintf(MOVE_CURSOR_FMT, y, x)
}

// SGR sets display attributes
// see: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func SGR(n int) string {
	return fmt.Sprintf(SGR_FMT, strconv.Itoa(n))
}

func SetColors(fg, bg int) string {
	colors := strings.Join([]string{
		strconv.Itoa(fg),
		strconv.Itoa(bg),
	}, ";")
	return fmt.Sprintf(SGR_FMT, colors)
}

func Erase(n int) string {
	return fmt.Sprintf(ERASE_FMT, n)
}

func SetMode(params ...string) string {
	return fmt.Sprintf(SET_MODE_FMT, strings.Join(params, ";"))
}

func ResetMode(params ...string) string {
	return fmt.Sprintf(RESET_MODE, strings.Join(params, ";"))
}

func ShowCursor() string {
	return SetMode(fmt.Sprintf("?%d", CURSOR))
}

func HideCursor() string {
	return ResetMode(fmt.Sprintf("?%d", CURSOR))
}
