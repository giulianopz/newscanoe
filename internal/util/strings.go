package util

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/exp/slices"
)

func RenderFeedRow(unreadCount, itemsLen int, name string) string {
	count := fmt.Sprintf("(%d/%d)", unreadCount, itemsLen)
	return fmt.Sprintf("%-20s %s", count, name)
}

var NoPubDate time.Time = time.Date(1001, 1, 1, 1, 1, 1, 1, time.UTC)

func RenderArticleRow(pubDate time.Time, title string) string {
	if pubDate == NoPubDate {
		return fmt.Sprintf("%-20s %s", "N/A", title)
	}
	return fmt.Sprintf("%-20s %s", pubDate.Format("2006-January-02"), title)
}

func IsLetter(input byte) bool {
	return (input >= 'A' && input <= 'Z') || (input >= 'a' && input <= 'z')
}

func IsDigit(input byte) bool {
	return input >= '0' && input <= '9'
}

var specialChars = []byte{
	';', ',', '/', '?', ':', '@', '&', '=', '+', '$', '-', '_', '.', '!', '~', '*', '(', ')', '#', '`',
}

/*
isSpecialChar returns if a given input char is a reserved char as per RFC 3986
see paragraph 2.2: https://www.ietf.org/rfc/rfc3986.txt
*/
func IsSpecialChar(input byte) bool {
	return slices.Contains(specialChars, input)
}

func PadToRight(s string, len int) string {
	sb := strings.Builder{}
	sb.WriteString(s)
	for i := len - utf8.RuneCountInString(s); i > 0; i-- {
		sb.WriteString(" ")
	}
	return sb.String()
}

func LineOfHyphens(width int) string {
	var sb strings.Builder
	for k := 0; k < width; k++ {
		sb.WriteString("-")
	}
	return sb.String()
}
