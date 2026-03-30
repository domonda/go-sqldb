package pqconn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
)

var (
	tableNameRegexp  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]{0,62}\.)?[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)
	columnNameRegexp = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,62}$`)

	reservedWords = map[string]struct{}{
		// Reserved key words
		"all":               {},
		"analyse":           {},
		"analyze":           {},
		"and":               {},
		"any":               {},
		"array":             {},
		"as":                {},
		"asc":               {},
		"asymmetric":        {},
		"authorization":     {},
		"between":           {},
		"binary":            {},
		"both":              {},
		"case":              {},
		"cast":              {},
		"check":             {},
		"collate":           {},
		"collation":         {},
		"column":            {},
		"concurrently":      {},
		"constraint":        {},
		"create":            {},
		"cross":             {},
		"current_catalog":   {},
		"current_date":      {},
		"current_role":      {},
		"current_schema":    {},
		"current_time":      {},
		"current_timestamp": {},
		"current_user":      {},
		"default":           {},
		"deferrable":        {},
		"desc":              {},
		"distinct":          {},
		"do":                {},
		"else":              {},
		"end":               {},
		"except":            {},
		"false":             {},
		"fetch":             {},
		"for":               {},
		"foreign":           {},
		"freeze":            {},
		"from":              {},
		"full":              {},
		"grant":             {},
		"group":             {},
		"having":            {},
		"ilike":             {},
		"in":                {},
		"initially":         {},
		"inner":             {},
		"intersect":         {},
		"into":              {},
		"is":                {},
		"isnull":            {},
		"join":              {},
		"lateral":           {},
		"leading":           {},
		"left":              {},
		"like":              {},
		"limit":             {},
		"localtime":         {},
		"localtimestamp":    {},
		"natural":           {},
		"not":               {},
		"notnull":           {},
		"null":              {},
		"offset":            {},
		"on":                {},
		"only":              {},
		"or":                {},
		"order":             {},
		"outer":             {},
		"overlaps":          {},
		"placing":           {},
		"primary":           {},
		"references":        {},
		"returning":         {},
		"right":             {},
		"select":            {},
		"session_user":      {},
		"similar":           {},
		"some":              {},
		"symmetric":         {},
		"table":             {},
		"tablesample":       {},
		"then":              {},
		"to":                {},
		"trailing":          {},
		"true":              {},
		"union":             {},
		"unique":            {},
		"user":              {},
		"using":             {},
		"variadic":          {},
		"verbose":           {},
		"when":              {},
		"where":             {},
		"window":            {},
		"with":              {},
		// Data types
		"anyelement":  {},
		"bigint":      {},
		"bigserial":   {},
		"bit":         {},
		"bool":        {},
		"boolean":     {},
		"box":         {},
		"bytea":       {},
		"char":        {},
		"character":   {},
		"cidr":        {},
		"circle":      {},
		"date":        {},
		"daterange":   {},
		"decimal":     {},
		"double":      {},
		"float4":      {},
		"float8":      {},
		"inet":        {},
		"int":         {},
		"int2":        {},
		"int4":        {},
		"int4range":   {},
		"int8":        {},
		"int8range":   {},
		"integer":     {},
		"interval":    {},
		"json":        {},
		"jsonb":       {},
		"line":        {},
		"lseg":        {},
		"macaddr":     {},
		"macaddr8":    {},
		"money":       {},
		"numeric":     {},
		"numrange":    {},
		"path":        {},
		"point":       {},
		"polygon":     {},
		"real":        {},
		"serial":      {},
		"serial2":     {},
		"serial4":     {},
		"serial8":     {},
		"smallint":    {},
		"smallserial": {},
		"text":        {},
		"timestamp":   {},
		"timestamptz": {},
		"timetz":      {},
		"tsquery":     {},
		"tsrange":     {},
		"tstzrange":   {},
		"tsvector":    {},
		"uuid":        {},
		"varchar":     {},
		"varying":     {},
		"void":        {},
		"xml":         {},
	}
)

// EscapeIdentifier wraps a PostgreSQL identifier in double-quotes when necessary,
// escaping any embedded double-quote characters as "".
// Quoting is applied when the identifier contains non-lowercase/non-underscore
// characters or is a PostgreSQL reserved word.
func EscapeIdentifier(ident string) string {
	// See https://doxygen.postgresql.org/ruleutils_8c.html#a8c18b3ffb8863e7740b32ef5f4c05ddc
	escaped := strings.ReplaceAll(ident, `"`, `""`)
	needsQuotes := len(escaped) != len(ident)
	if !needsQuotes {
		for _, r := range ident {
			if (r < 'a' || r > 'z') && r != '_' {
				needsQuotes = true
				break
			}
		}
	}
	if !needsQuotes {
		_, needsQuotes = reservedWords[strings.ToLower(ident)]
	}
	if needsQuotes {
		return `"` + escaped + `"`
	}
	return ident
}

// QueryFormatter is the [sqldb.QueryFormatter] implementation for PostgreSQL.
// Uses double-quote identifier escaping, $N placeholders, and standard single-quote string literals.
type QueryFormatter struct{}

// FormatTableName implements [sqldb.QueryFormatter.FormatTableName].
func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	if schema, table, ok := strings.Cut(name, "."); ok {
		return EscapeIdentifier(schema) + "." + EscapeIdentifier(table), nil
	}
	return EscapeIdentifier(name), nil
}

// FormatColumnName implements [sqldb.QueryFormatter.FormatColumnName].
func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return EscapeIdentifier(name), nil
}

// FormatPlaceholder implements [sqldb.QueryFormatter.FormatPlaceholder].
func (f QueryFormatter) FormatPlaceholder(paramIndex int) string {
	return "$" + strconv.Itoa(paramIndex+1)
}

// FormatStringLiteral implements [sqldb.QueryFormatter.FormatStringLiteral].
func (QueryFormatter) FormatStringLiteral(str string) string {
	return pq.QuoteLiteral(str)
}

// MaxArgs implements [sqldb.QueryFormatter.MaxArgs].
func (QueryFormatter) MaxArgs() int {
	return 65535
}
