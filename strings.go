package sqldb

import (
	"strings"
	"unicode"

	"github.com/corazawaf/libinjection-go"
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

// IsSQLInjection checks if the given string contains SQL injection patterns
// using libinjection-go. It detects common SQL injection attempts including
// boolean-based injections, UNION queries, comment-based bypasses, stacked
// queries, and blind injection techniques.
//
// Returns true if SQL injection is detected, along with diagnostic information
// about the detected pattern. Returns false for legitimate strings that may
// contain SQL-like characters (e.g., names with apostrophes like "O'Brien").
func IsSQLInjection(str string) (is bool, info string) {
	return libinjection.IsSQLi(str)
}
