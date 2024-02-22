package display

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/giulianopz/newscanoe/internal/ansi"
)

func (d *display) writeLineAt(line string, y int) {
	slog.Info("writing", "line", line, "y", y)
	fmt.Fprint(os.Stdout, ansi.MoveCursor(y, 1))
	fmt.Fprint(os.Stdout, ansi.EraseToEndOfLine(ansi.ERASE_ENTIRE_LINE))
	fmt.Fprint(os.Stdout, ansi.SGR(ansi.REVERSE_COLOR))
	fmt.Fprint(os.Stdout, line)
	fmt.Fprint(os.Stdout, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))
}
