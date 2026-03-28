package oraconn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/domonda/go-sqldb"
)

var (
	// Oracle identifiers can be up to 128 chars (since Oracle 12c R2).
	// Schema-qualified: schema.table where each part is up to 128 chars.
	tableNameRegexp  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_#$]{0,127}\.)?[a-zA-Z_][a-zA-Z0-9_#$]{0,127}$`)
	columnNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_#$]{0,127}$`)

	// Common Oracle reserved words that require double-quote escaping.
	// https://docs.oracle.com/en/database/oracle/oracle-database/23/sqlrf/Oracle-SQL-Reserved-Words.html
	reservedWords = map[string]struct{}{
		"access":     {},
		"add":        {},
		"all":        {},
		"alter":      {},
		"and":        {},
		"any":        {},
		"as":         {},
		"asc":        {},
		"audit":      {},
		"between":    {},
		"by":         {},
		"case":       {},
		"char":       {},
		"check":      {},
		"cluster":    {},
		"column":     {},
		"comment":    {},
		"compress":   {},
		"connect":    {},
		"create":     {},
		"cross":      {},
		"current":    {},
		"date":       {},
		"decimal":    {},
		"default":    {},
		"delete":     {},
		"desc":       {},
		"distinct":   {},
		"drop":       {},
		"else":       {},
		"end":        {},
		"except":     {},
		"exclusive":  {},
		"exists":     {},
		"false":      {},
		"fetch":      {},
		"file":       {},
		"float":      {},
		"for":        {},
		"from":       {},
		"full":       {},
		"grant":      {},
		"group":      {},
		"having":     {},
		"identified": {},
		"immediate":  {},
		"in":         {},
		"increment":  {},
		"index":      {},
		"initial":    {},
		"inner":      {},
		"insert":     {},
		"integer":    {},
		"intersect":  {},
		"into":       {},
		"is":         {},
		"join":       {},
		"left":       {},
		"level":      {},
		"like":       {},
		"lock":       {},
		"long":       {},
		"maxextents": {},
		"merge":      {},
		"minus":      {},
		"mlslabel":   {},
		"mode":       {},
		"modify":     {},
		"natural":    {},
		"noaudit":    {},
		"nocompress": {},
		"not":        {},
		"nowait":     {},
		"null":       {},
		"number":     {},
		"of":         {},
		"offline":    {},
		"offset":     {},
		"on":         {},
		"online":     {},
		"option":     {},
		"or":         {},
		"order":      {},
		"outer":      {},
		"pctfree":    {},
		"prior":      {},
		"public":     {},
		"raw":        {},
		"rename":     {},
		"resource":   {},
		"returning":  {},
		"revoke":     {},
		"right":      {},
		"row":        {},
		"rowid":      {},
		"rownum":     {},
		"rows":       {},
		"select":     {},
		"session":    {},
		"set":        {},
		"share":      {},
		"size":       {},
		"smallint":   {},
		"start":      {},
		"successful": {},
		"synonym":    {},
		"sysdate":    {},
		"table":      {},
		"then":       {},
		"to":         {},
		"trigger":    {},
		"true":       {},
		"uid":        {},
		"union":      {},
		"unique":     {},
		"update":     {},
		"user":       {},
		"validate":   {},
		"values":     {},
		"varchar":    {},
		"varchar2":   {},
		"view":       {},
		"whenever":   {},
		"where":      {},
		"with":       {},
		// Common data types
		"binary_double": {},
		"binary_float":  {},
		"blob":          {},
		"boolean":       {},
		"clob":          {},
		"int":           {},
		"nchar":         {},
		"nclob":         {},
		"numeric":       {},
		"nvarchar2":     {},
		"real":          {},
		"timestamp":     {},
		"xml":           {},
		"xmltype":       {},
	}
)

// EscapeIdentifier wraps an Oracle identifier in double quotes when necessary,
// escaping any embedded double-quote characters as "".
// Quoting is applied when the identifier contains non-uppercase/non-underscore
// characters or is an Oracle reserved word.
// Oracle treats unquoted identifiers as uppercase, so any identifier with
// lowercase letters is also quoted.
func EscapeIdentifier(ident string) string {
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

// QueryFormatter is the [sqldb.QueryFormatter] implementation for Oracle.
// Uses double-quote identifier escaping, :N placeholders, and standard single-quote string literals.
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
func (QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return ":" + strconv.Itoa(paramIndex+1)
}

// FormatStringLiteral implements [sqldb.QueryFormatter.FormatStringLiteral].
func (QueryFormatter) FormatStringLiteral(str string) string {
	return sqldb.QuoteStringLiteral(str)
}

// MaxArgs implements [sqldb.QueryFormatter.MaxArgs].
func (QueryFormatter) MaxArgs() int {
	return 65535
}
