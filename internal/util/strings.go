package util

import (
	"strings"
	"unicode/utf8"

	"golang.org/x/exp/slices"
)

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

func PadToLeft(s string, len int) string {
	sb := strings.Builder{}
	for i := len - utf8.RuneCountInString(s); i > 0; i-- {
		sb.WriteString(" ")
	}
	sb.WriteString(s)
	return sb.String()
}

func LineOf(width int, symbol string) string {
	var sb strings.Builder
	for k := 0; k < width; k++ {
		sb.WriteString(symbol)
	}
	return sb.String()
}
