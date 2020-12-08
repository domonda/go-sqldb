package sqldb

import "strings"

// SanitizeString returns valid UTF-8 without zero byte characters.
func SanitizeString(s string) string {
	return strings.ReplaceAll(strings.ToValidUTF8(s, ""), "\x00", "")
}
