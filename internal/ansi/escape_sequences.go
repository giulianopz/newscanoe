package ansi

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/giulianopz/newscanoe/internal/ascii"
)

/*
see: https://vt100.net/docs/vt100-ug/chapter3.html
*/

const (

	// control sequence introducer
	csi = string(ascii.ESC) + "["

	MOVE_CURSOR_FMT = csi + "%d;%dH"
	SGR_FMT         = csi + "%sm"
	ERASE_EOS_FMT   = csi + "%dJ"
	ERASE_EOL_FMT   = csi + "%dK"
	SET_MODE_FMT    = csi + "%sh"
	RESET_MODE      = csi + "%sl"
)

func MoveCursor(y, x int) string {
	return fmt.Sprintf(MOVE_CURSOR_FMT, y, x)
}

// SGR sets display attributes
// see: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func SGR(nums ...int) string {
	params := make([]string, 0)
	for _, n := range nums {
		params = append(params, strconv.Itoa(n))
	}
	return fmt.Sprintf(SGR_FMT, strings.Join(params, ";"))
}

func WhiteFG() string {
	return SGR(38, 2, 255, 255, 255)
}

func EraseToEndOfScreen(n int) string {
	return fmt.Sprintf(ERASE_EOS_FMT, n)
}

func EraseToEndOfLine(n int) string {
	return fmt.Sprintf(ERASE_EOL_FMT, n)
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
