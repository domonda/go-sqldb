package sqldb

import (
	"strings"
	"unicode"
)

// SanitizeString returns valid UTF-8 without any control code characters.
func SanitizeString(s string) string {
	return strings.Map(removeControlCodes, strings.ToValidUTF8(s, ""))
}

func removeControlCodes(r rune) rune {
	if unicode.IsControl(r) {
		return -1
	}
	return r
}
