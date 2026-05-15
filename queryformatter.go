package sqldb

import (
	"fmt"
	"regexp"
	"strconv"
)

// QueryFormatter has methods for formatting parts
// of a query dependent on the database driver.
type QueryFormatter interface {
	// FormatTableName formats the name of a table or view
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

	// MaxArgs returns the maximum number of query parameters
	// supported by the database driver per query.
	MaxArgs() int

	// SubstitutePlaceholders substitutes argument placeholders in the query
	// with formatted values, returning a human-readable SQL string.
	// Intended for debugging or logging — not for execution.
	// The implementation decides if any substitution is done
	// or if the query is returned unchanged to not leak args to error logs.
	SubstitutePlaceholders(query string, args []any) (string, error)
}

var _ QueryFormatter = StdQueryFormatter{}

// StdQueryFormatter is a [QueryFormatter] implementation that validates
// identifiers with a conservative regex ([a-zA-Z_][a-zA-Z0-9_]* with optional
// schema prefix for tables), returns string literals using standard single-quote
// doubling without backslash escaping (ANSI SQL compliant), and produces either
// uniform `?` placeholders or positional placeholders with a configurable prefix
// (e.g. `$` for PostgreSQL, `@p` for SQL Server, `:` for Oracle).
// It does not quote identifiers; use a database-specific [QueryFormatter] when
// identifier quoting for reserved words is required.
type StdQueryFormatter struct {
	// PlaceholderPosPrefix is prefixed before
	// the one based placeholder position number in queries.
	// If empty, the default placeholder `?` is used
	// without a position number.
	//
	// Positional placeholder formats for different databases:
	//   MySQL: ?
	//   SQLite: ?
	//   PostgreSQL: $1, $2, ...
	//   SQL Server: @p1, @p2, ...
	//   Oracle: :1, :2, ...
	PlaceholderPosPrefix string

	// DisableSubstitutePlaceholders if true, the query is
	// returned unchanged from the SubstitutePlaceholders method
	// disabling its default behavior of substituting placeholders with values.
	DisableSubstitutePlaceholders bool
}

// NewQueryFormatter returns a [StdQueryFormatter] with the given `placeholderPosPrefix`.
func NewQueryFormatter(placeholderPosPrefix string) StdQueryFormatter {
	return StdQueryFormatter{PlaceholderPosPrefix: placeholderPosPrefix}
}

var (
	stdTableNameRegexp  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]*\.)?[a-zA-Z_][a-zA-Z0-9_]*$`)
	stdColumnNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// FormatTableName implements the QueryFormatter interface.
func (StdQueryFormatter) FormatTableName(name string) (string, error) {
	if !stdTableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	return name, nil
}

// FormatColumnName implements the QueryFormatter interface.
func (StdQueryFormatter) FormatColumnName(name string) (string, error) {
	if !stdColumnNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return name, nil
}

// FormatPlaceholder implements the QueryFormatter interface.
func (f StdQueryFormatter) FormatPlaceholder(paramIndex int) string {
	if f.PlaceholderPosPrefix == "" {
		return "?"
	}
	return f.PlaceholderPosPrefix + strconv.Itoa(paramIndex+1)
}

// FormatStringLiteral implements the QueryFormatter interface.
func (StdQueryFormatter) FormatStringLiteral(str string) string {
	return QuoteStringLiteral(str)
}

// MaxArgs implements the QueryFormatter interface.
func (StdQueryFormatter) MaxArgs() int {
	return 65535
}

// SubstitutePlaceholders implements the QueryFormatter interface.
func (f StdQueryFormatter) SubstitutePlaceholders(query string, args []any) (string, error) {
	if f.DisableSubstitutePlaceholders {
		return query, nil
	}
	return SubstitutePlaceholders(f, query, args)
}
