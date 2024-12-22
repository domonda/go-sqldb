package sqldb

import (
	"fmt"
	"regexp"
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
}

var stdNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type StdQueryFormatter struct {
	// PlaceholderFmt is a format string for positional
	// parameter placeholders in queries.
	// The format string must contain a single `%d`
	// for the parameter index starting at one.
	// If empty, the default placeholder `?` is used.
	//
	// Positional placeholder formats for different databases:
	//   MySQL: ?
	//   SQLite: ?
	//   PostgreSQL: $1, $2, ...
	//   SQL Server: @p1, @p2, ...
	//   Oracle: :1, :2, ...
	PlaceholderFmt string
}

func (StdQueryFormatter) FormatTableName(name string) (string, error) {
	if !stdNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	return name, nil
}

func (StdQueryFormatter) FormatColumnName(name string) (string, error) {
	if !stdNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return name, nil
}

func (f StdQueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	if f.PlaceholderFmt != "" {
		return fmt.Sprintf(f.PlaceholderFmt, paramIndex+1)
	}
	return "?"
}
