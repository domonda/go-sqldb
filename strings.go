package sqldb

import (
	"strings"
	"unicode"
)

// SanitizeString returns valid UTF-8 only with printable characters.
func SanitizeString(s string) string {
	return strings.Map(
		func(r rune) rune {
			if r == '�' || !unicode.IsPrint(r) {
				return -1
			}
			return r
		},
		s,
	)
}
