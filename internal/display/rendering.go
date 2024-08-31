package display

import (
	"fmt"
	"log"
	"time"

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

func (d *display) renderArticleText(raw [][]byte) {

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

	d.rows = make([][]*cell, 0)
	line := make([]*cell, 0)
	for _, c := range runes {

		if c == '\r' || c == '\n' {

			if len(line) != 0 {
				d.rows = append(d.rows, add(margin, line))
			}

			d.rows = append(d.rows, []*cell{})

			line = make([]*cell, 0)

			continue
		}

		if c == '\t' {
			for i := 0; i < 4; i++ {
				line = append(line, textcell(' '))
			}
			continue
		}

		if len(line) < textSpace {
			line = append(line, textcell(c))
		} else {

			if line[len(line)-1].char != ' ' {

				var lastIdx int = len(line)
				tmp := make([]*cell, 0)
				for i := lastIdx - 1; i >= 0; i-- {
					lastIdx--
					if line[i].char != ' ' {
						tmp = append([]*cell{line[i]}, tmp...)
					} else {
						break
					}
				}

				d.rows = append(d.rows, add(margin, line[:lastIdx]))
				line = tmp
			} else {

				d.rows = append(d.rows, add(margin, line))
				line = make([]*cell, 0)
			}

			line = append(line, textcell(c))
		}
	}

	if len(line) != 0 {
		d.rows = append(d.rows, add(margin, line))
	}
}

func add(margin int, line []*cell) []*cell {
	if margin != 0 {
		padded := make([]*cell, 0)
		for margin != 0 {
			padded = append(padded, textcell(' '))
			margin--
		}
		padded = append(padded, line...)
		return padded
	}
	return line
}
