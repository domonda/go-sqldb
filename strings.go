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

// SanitizeStringTrimSpace returns valid UTF-8 only with printable characters
// with leading and trailing whitespace trimmed away.
func SanitizeStringTrimSpace(s string) string {
	return strings.TrimSpace(SanitizeString(s))
}
