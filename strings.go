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
//
// This function is intended to be called by application code at system
// boundaries — for example, to validate a user-supplied string before using
// it as a dynamic table name, column name, or filter value. It should not be
// applied to developer-authored SQL fragments (WHERE clauses, RETURNING
// clauses, etc.) that intentionally contain SQL syntax, as those would
// produce false positives. go-sqldb's primary injection defenses are
// parameterized queries for values and regex validation for identifiers.
func IsSQLInjection(str string) (is bool, info string) {
	if is, info = libinjection.IsSQLi(str); is {
		if info == "" {
			info = "sqli"
		}
		return true, info
	}
	// libinjection misses MySQL hash comments after a quote (admin' #)
	// and stacked DDL queries without a leading quote (; DROP TABLE users--)
	upper := strings.ToUpper(str)
	for i, c := range upper {
		switch c {
		case '\'':
			rest := strings.TrimLeft(upper[i+1:], " \t")
			if strings.HasPrefix(rest, "#") {
				return true, "hash comment after string terminator"
			}
		case ';':
			rest := strings.TrimLeft(upper[i+1:], " \t")
			for _, kw := range []string{"DROP ", "TRUNCATE ", "ALTER ", "CREATE ", "EXEC "} {
				if strings.HasPrefix(rest, kw) {
					return true, "stacked " + strings.ToLower(strings.TrimRight(kw, " ")) + " query"
				}
			}
		}
	}
	return false, ""
}
