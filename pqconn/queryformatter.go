package pqconn

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/domonda/go-sqldb"
	"github.com/domonda/go-sqldb/db"
)

var (
	tableNameRegexp = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_]{0,62}\.)?[a-zA-Z_][a-zA-Z0-9_]{0,62}$`)
	columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][0-9a-zA-Z_]{0,58}$`)

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
		"both":              {},
		"case":              {},
		"cast":              {},
		"check":             {},
		"collate":           {},
		"column":            {},
		"constraint":        {},
		"create":            {},
		"current_catalog":   {},
		"current_date":      {},
		"current_role":      {},
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
		"from":              {},
		"grant":             {},
		"group":             {},
		"having":            {},
		"in":                {},
		"initially":         {},
		"intersect":         {},
		"into":              {},
		"lateral":           {},
		"leading":           {},
		"limit":             {},
		"localtime":         {},
		"localtimestamp":    {},
		"not":               {},
		"null":              {},
		"offset":            {},
		"on":                {},
		"only":              {},
		"or":                {},
		"order":             {},
		"placing":           {},
		"primary":           {},
		"references":        {},
		"returning":         {},
		"select":            {},
		"session_user":      {},
		"some":              {},
		"symmetric":         {},
		"table":             {},
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

type QueryFormatter struct{}

func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	if schema, table, ok := strings.Cut(name, "."); ok {
		return EscapeIdentifier(schema) + "." + EscapeIdentifier(table), nil
	}
	return EscapeIdentifier(name), nil
}

func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	return EscapeIdentifier(name), nil
}

func (f QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "$" + strconv.Itoa(paramIndex+1)
}

func (QueryFormatter) FormatStringLiteral(str string) string {
	return sqldb.FormatSingleQuoteStringLiteral(str)
}

func NewTypeMapper() *db.StagedTypeMapper {
	return &db.StagedTypeMapper{
		Types: map[reflect.Type]string{
			reflect.TypeFor[time.Time](): "timestamptz",
		},
		Kinds: map[reflect.Kind]string{
			reflect.Bool:    "boolean",
			reflect.Int:     "bigint",
			reflect.Int8:    "smallint",
			reflect.Int16:   "smallint",
			reflect.Int32:   "integer",
			reflect.Int64:   "bigint",
			reflect.Uint:    "bigint",
			reflect.Uint8:   "smallint",
			reflect.Uint16:  "integer",
			reflect.Uint32:  "bigint",
			reflect.Uint64:  "bigint", // 64 unsigned integer does not fit completely into signed bigint
			reflect.Float32: "float4",
			reflect.Float64: "float8",
			reflect.String:  "text",
		},
	}
}
