package bar

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"github.com/giulianopz/newscanoe/internal/ansi"
	"github.com/giulianopz/newscanoe/internal/util"
)

type Bar struct {
	leftText, rightText string
}

func NewBar() *Bar {
	return &Bar{}
}

func (bb *Bar) SetText(l, r string) {
	bb.leftText = l
	bb.rightText = r
}

func (bb *Bar) Build(width int) string {
	buf := &bytes.Buffer{}
	fmt.Fprint(buf, ansi.SGR(ansi.REVERSE_COLOR))

	var text string

	barTextRunes := utf8.RuneCountInString(bb.leftText + bb.rightText)
	if width > barTextRunes {
		text = util.PadToRight(bb.leftText, width-utf8.RuneCountInString(bb.rightText)) + bb.rightText
	} else if utf8.RuneCountInString(bb.rightText) <= width {
		text = util.PadToLeft(bb.rightText, width)
	} else {
		text = util.LineOf(width, " ")
	}

	fmt.Fprint(buf, text)

	fmt.Fprint(buf, ansi.SGR(ansi.ALL_ATTRIBUTES_OFF))

	return buf.String()
}
