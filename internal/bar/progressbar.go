package bar

import (
	"fmt"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/internal/ansi"
)

const (
	progressPercentageFmt = "Progress: [%3d%%]"
	progressBarFmt        = "[%s]"

	hashMark = 35
	point    = 46
)

var (
	allocated = utf8.RuneCountInString(fmt.Sprintf(progressPercentageFmt+" "+progressBarFmt, 100, ""))
)

// ProgressBar is a progress bar with (Debian) APT style
type ProgressBar struct {
	mu sync.Mutex

	// max elements to consider
	max int
	// current element
	k int
	// percentage of progress
	percentage int
	// percentage inside text bar
	limit int

	y, x, width int
	// free space left to write the progress bar
	free int

	bar []byte
}

func NewProgressBar(y, x, width, max int) *ProgressBar {
	free := width - allocated
	bar := make([]byte, free)
	for i := 0; i < free; i++ {
		bar[i] = point
	}

	return &ProgressBar{
		y:     y,
		x:     x,
		width: width,
		max:   max,
		free:  free,
		bar:   bar,
	}
}

func (pb *ProgressBar) IncrByOne() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	pb.k++

	// k : max = percentage : 100
	pb.percentage = (pb.k * 100) / pb.max
	// scale percentage to index the progres bar array
	pb.limit = (pb.k*pb.free)/pb.max - 1

	for i := 0; i <= pb.limit; i++ {
		pb.bar[i] = hashMark
	}

	pb.printCurrentState()
}

func (pb *ProgressBar) printCurrentState() {
	fmt.Fprint(os.Stdout, ansi.MoveCursor(pb.y, pb.x))
	fmt.Fprint(os.Stdout, ansi.EraseToEndOfLine(2))
	fmt.Fprint(os.Stdout, ansi.SGR(ansi.REVERSE_COLOR))

	percentage := fmt.Sprintf(progressPercentageFmt, pb.percentage)
	bar := fmt.Sprintf(progressBarFmt, string(pb.bar))

	fmt.Fprintf(os.Stdout, "%s %s", percentage, bar)
}
