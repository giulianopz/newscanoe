package util

import (
	"fmt"
	"time"

	"golang.org/x/exp/slices"
)

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
