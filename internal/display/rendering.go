package display

import (
	"fmt"
	"log"
	"time"

	"github.com/giulianopz/newscanoe/internal/ansi"
	"github.com/giulianopz/newscanoe/internal/feed"
)

func feedRow(unreadCount, itemsLen int, name string) string {
	count := fmt.Sprintf("(%d/%d)", unreadCount, itemsLen)
	return fmt.Sprintf("%-20s %s", count, name)
}

func articleRow(pubDate time.Time, title string) string {
	if pubDate == feed.NoPubDate {
		return fmt.Sprintf("%-20s %s", "N/A", title)
	}
	return fmt.Sprintf("%-20s %s", pubDate.Format("2006-January-02"), title)
}

func (d *display) reflow(raw [][]byte) {

	log.Default().Println("width: ", d.width)

	var textSpace, margin int = d.width - 1, 0

	if d.width > 100 {
		textSpace = (d.width - 1) / 4 * 2
		margin = ((d.width - 1) - textSpace) / 2
	}

	runes := make([]rune, 0)
	for _, row := range raw {
		if len(row) == 0 {
			runes = append(runes, '\n')
		}
		for _, c := range string(row) {
			runes = append(runes, c)
		}
	}

	var (
		csiNotClosed bool
		csi          string
	)

	anySplittedSeq := func(line []*cell) (current []*cell, next []*cell) {
		if csiNotClosed {
			return append(line, stringToCells(ansi.AllAttributesOff)...), stringToCells(csi)
		}
		return line, []*cell{}
	}

	d.rows = make([][]*cell, 0)
	line := make([]*cell, 0)

	for i := 0; i < len(runes); i++ {

		c := runes[i]

		// skip ANSI escape sequences
		if c == '\x1b' && (i+1) < len(runes)-1 && runes[i+1] == '[' {

			csiNotClosed = true
			csi += string(c)

			for !ansi.Cst(c) {
				line = append(line, runeToCell(c))

				i++
				c = runes[i]
				csi += string(c)
			}

			if csi == ansi.AllAttributesOff {
				csiNotClosed = false
			}
		}

		// preserve original new lines
		if c == '\r' || c == '\n' {
			if len(line) != 0 {
				current, next := anySplittedSeq(line)
				d.appendCells(add(margin, current))
				line = next
			} else {
				d.appendCells([]*cell{})
				line = make([]*cell, 0)
			}
			continue
		}

		// convert a tab to four whitespaces
		if c == '\t' {
			for i := 0; i < 4; i++ {
				line = append(line, runeToCell(' '))
			}
			continue
		}

		// append current char if current line is within the window width
		if charslen(line) < textSpace {
			line = append(line, runeToCell(c))
			continue
		}
		// otherwise, before appending the current char, break current line
		lll := cellsToString(line)
		log.Default().Println(lll)
		// if we are at a word boundary, just append the current line
		if line[len(line)-1].char == ' ' {
			current, next := anySplittedSeq(line)
			d.appendCells(add(margin, current))
			line = next
		} else { // go back to the previous word boundary and break the line

			var lastIdx int = len(line)
			afterWhiteSpace := make([]*cell, 0)
			for i := lastIdx - 1; i >= 0; i-- {
				lastIdx--
				if line[i].char == ' ' {
					break
				}
				afterWhiteSpace = append([]*cell{line[i]}, afterWhiteSpace...)
			}

			beforeWhiteSpace := line[:lastIdx]

			current, next := anySplittedSeq(beforeWhiteSpace)
			d.appendCells(add(margin, current))
			line = append(next, afterWhiteSpace...)
		}

		line = append(line, runeToCell(c))
	}

	// append last line (if any)
	if len(line) != 0 {
		current, _ := anySplittedSeq(line)
		d.appendCells(add(margin, current))
	}
}

func add(margin int, line []*cell) []*cell {
	if margin != 0 {
		padded := make([]*cell, 0)
		for margin != 0 {
			padded = append(padded, runeToCell(' '))
			margin--
		}
		padded = append(padded, line...)
		return padded
	}
	return line
}
