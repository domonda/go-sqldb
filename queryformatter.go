package sqldb

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// QueryFormatter has methods for formatting parts
// of a query dependent on the database driver.
type QueryFormatter interface {
	// FormatTableName formats the name of a name or view
	// using quotes or escape characters if necessary.
	FormatTableName(name string) (string, error)

	// FormatColumnName formats the name of a column
	// using quotes or escape characters if necessary.
	FormatColumnName(name string) (string, error)

	// FormatPlaceholder formats a query parameter placeholder
	// for the paramIndex starting at zero.
	FormatPlaceholder(paramIndex int) string

	// FormatStringLiteral formats a string literal
	// by adding quotes or escaping characters if necessary.
	FormatStringLiteral(str string) string
}

type StdQueryFormatter struct {
	// PlaceholderPosPrefix is prefixed before
	// the one based placeholder position number in queries.
	// If empty, the default placeholder `?` is used
	// witout a position number is used.
	//
	// Positional placeholder formats for different databases:
	//   MySQL: ?
	//   SQLite: ?
	//   PostgreSQL: $1, $2, ...
	//   SQL Server: @p1, @p2, ...
	//   Oracle: :1, :2, ...
	PlaceholderPosPrefix string
}

func NewStdQueryFormatter(placeholderPosPrefix string) StdQueryFormatter {
	return StdQueryFormatter{PlaceholderPosPrefix: placeholderPosPrefix}
}

var (
	stdTableNameRegexp  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*\.)?[a-zA-Z_][a-zA-Z0-9_]*$`)
	stdColumnNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

func (StdQueryFormatter) FormatTableName(name string) (string, error) {
	if !stdTableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	return name, nil
}

func (StdQueryFormatter) FormatColumnName(name string) (string, error) {
	if !stdColumnNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return name, nil
}

func (f StdQueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	if f.PlaceholderPosPrefix == "" {
		return "?"
	}
	return f.PlaceholderPosPrefix + strconv.Itoa(paramIndex+1)
}

func (StdQueryFormatter) FormatStringLiteral(str string) string {
	return FormatSingleQuoteStringLiteral(str)
}

func FormatSingleQuoteStringLiteral(str string) string {
	quoted := len(str) >= 2 && str[0] == '\'' && str[len(str)-1] == '\''
	if quoted {
		inner := str[1 : len(str)-1]
		if !strings.Contains(inner, `'`) {
			return str // fast path
		}
		str = inner
	}
	var b strings.Builder
	b.Grow(len(str) + 4) // 2 quotes plus some escaping
	b.WriteByte('\'')
	for i, r := range str {
		switch r {
		case '\\':
			nextIsQuote := i+1 < len(str) && str[i+1] == '\''
			if nextIsQuote {
				// Change `\'` to `''`
				b.WriteByte('\'')
				continue
			}
			b.WriteByte('\\')

		case '\'':
			lastWasEscape := i > 0 && (str[i-1] == '\\' || str[i-1] == '\'')
			if lastWasEscape {
				// Already in escape sequence
				b.WriteByte('\'')
				continue
			}
			nextIsQuote := i+1 < len(str) && str[i+1] == '\''
			if nextIsQuote {
				// First of two quotes
				b.WriteByte('\'')
				continue
			}
			// Escape quote
			b.WriteString(`''`)

		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
