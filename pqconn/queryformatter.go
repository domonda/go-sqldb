package pqconn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

type QueryFormatter struct{}

func (QueryFormatter) FormatTableName(name string) (string, error) {
	// See https://doxygen.postgresql.org/ruleutils_8c.html#a8c18b3ffb8863e7740b32ef5f4c05ddc
	if !tableNameRegexp.MatchString(name) {
		return "", fmt.Errorf("invalid table name %q", name)
	}
	if _, reserved := reservedWords[strings.ToLower(name)]; reserved {
		return `"` + name + `"`, nil
	}
	return name, nil
}

func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid column name %q", name)
	}
	if _, reserved := reservedWords[strings.ToLower(name)]; reserved {
		return `"` + name + `"`, nil
	}
	return name, nil
}

func (f QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "$" + strconv.Itoa(paramIndex+1)
}
