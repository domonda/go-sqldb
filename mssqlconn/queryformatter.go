package mssqlconn

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/domonda/go-sqldb"
)

// https://learn.microsoft.com/en-us/sql/relational-databases/databases/database-identifiers
var (
	// MSSQL identifiers can be up to 128 chars.
	// Schema-qualified: schema.table where each part is up to 128 chars.
	tableNameRegex  = regexp.MustCompile(`^([a-zA-Z_][a-zA-Z0-9_ ]{0,127}\.)?[a-zA-Z_][a-zA-Z0-9_ ]{0,127}$`)
	columnNameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_ ]{0,127}$`)

	// Common T-SQL reserved words that require bracket escaping.
	// https://learn.microsoft.com/en-us/sql/t-sql/language-elements/reserved-keywords-transact-sql
	reservedWords = map[string]struct{}{
		"add":          {},
		"all":          {},
		"alter":        {},
		"and":          {},
		"any":          {},
		"as":           {},
		"asc":          {},
		"authorization": {},
		"backup":       {},
		"begin":        {},
		"between":      {},
		"break":        {},
		"browse":       {},
		"bulk":         {},
		"by":           {},
		"cascade":      {},
		"case":         {},
		"check":        {},
		"checkpoint":   {},
		"close":        {},
		"clustered":    {},
		"coalesce":     {},
		"collate":      {},
		"column":       {},
		"commit":       {},
		"compute":      {},
		"constraint":   {},
		"contains":     {},
		"continue":     {},
		"convert":      {},
		"create":       {},
		"cross":        {},
		"current":      {},
		"current_date": {},
		"current_time": {},
		"current_timestamp": {},
		"current_user": {},
		"cursor":       {},
		"database":     {},
		"dbcc":         {},
		"deallocate":   {},
		"declare":      {},
		"default":      {},
		"delete":       {},
		"deny":         {},
		"desc":         {},
		"disk":         {},
		"distinct":     {},
		"distributed":  {},
		"double":       {},
		"drop":         {},
		"dump":         {},
		"else":         {},
		"end":          {},
		"errlvl":       {},
		"escape":       {},
		"except":       {},
		"exec":         {},
		"execute":      {},
		"exists":       {},
		"exit":         {},
		"external":     {},
		"fetch":        {},
		"file":         {},
		"fillfactor":   {},
		"for":          {},
		"foreign":      {},
		"freetext":     {},
		"from":         {},
		"full":         {},
		"function":     {},
		"goto":         {},
		"grant":        {},
		"group":        {},
		"having":       {},
		"holdlock":     {},
		"identity":     {},
		"if":           {},
		"in":           {},
		"index":        {},
		"inner":        {},
		"insert":       {},
		"intersect":    {},
		"into":         {},
		"is":           {},
		"join":         {},
		"key":          {},
		"kill":         {},
		"left":         {},
		"like":         {},
		"lineno":       {},
		"load":         {},
		"merge":        {},
		"national":     {},
		"nocheck":      {},
		"nonclustered": {},
		"not":          {},
		"null":         {},
		"nullif":       {},
		"of":           {},
		"off":          {},
		"offsets":      {},
		"on":           {},
		"open":         {},
		"option":       {},
		"or":           {},
		"order":        {},
		"outer":        {},
		"over":         {},
		"percent":      {},
		"pivot":        {},
		"plan":         {},
		"precision":    {},
		"primary":      {},
		"print":        {},
		"proc":         {},
		"procedure":    {},
		"public":       {},
		"raiserror":    {},
		"read":         {},
		"readtext":     {},
		"reconfigure":  {},
		"references":   {},
		"replication":  {},
		"restore":      {},
		"restrict":     {},
		"return":       {},
		"revert":       {},
		"revoke":       {},
		"right":        {},
		"rollback":     {},
		"rowcount":     {},
		"rowguidcol":   {},
		"rule":         {},
		"save":         {},
		"schema":       {},
		"select":       {},
		"session_user": {},
		"set":          {},
		"setuser":      {},
		"shutdown":     {},
		"some":         {},
		"statistics":   {},
		"system_user":  {},
		"table":        {},
		"tablesample":  {},
		"textsize":     {},
		"then":         {},
		"to":           {},
		"top":          {},
		"tran":         {},
		"transaction":  {},
		"trigger":      {},
		"truncate":     {},
		"try_convert":  {},
		"tsequal":      {},
		"union":        {},
		"unique":       {},
		"unpivot":      {},
		"update":       {},
		"updatetext":   {},
		"use":          {},
		"user":         {},
		"values":       {},
		"varying":      {},
		"view":         {},
		"waitfor":      {},
		"when":         {},
		"where":        {},
		"while":        {},
		"with":         {},
		"writetext":    {},
		// Common data types
		"bigint":     {},
		"binary":     {},
		"bit":        {},
		"char":       {},
		"date":       {},
		"datetime":   {},
		"datetime2":  {},
		"decimal":    {},
		"float":      {},
		"geography":  {},
		"geometry":   {},
		"image":      {},
		"int":        {},
		"money":      {},
		"nchar":      {},
		"ntext":      {},
		"numeric":    {},
		"nvarchar":   {},
		"real":       {},
		"smallint":   {},
		"smallmoney": {},
		"sql_variant": {},
		"text":       {},
		"time":       {},
		"timestamp":  {},
		"tinyint":    {},
		"uniqueidentifier": {},
		"varbinary":  {},
		"varchar":    {},
		"xml":        {},
	}
)

// EscapeIdentifier wraps a MSSQL identifier in [brackets],
// escaping any ] characters as ]].
func EscapeIdentifier(ident string) string {
	escaped := strings.ReplaceAll(ident, "]", "]]")
	needsBrackets := len(escaped) != len(ident)
	if !needsBrackets {
		for _, r := range ident {
			if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
				needsBrackets = true
				break
			}
		}
	}
	if !needsBrackets {
		_, needsBrackets = reservedWords[strings.ToLower(ident)]
	}
	if needsBrackets {
		return "[" + escaped + "]"
	}
	return ident
}

type QueryFormatter struct{}

func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegex.MatchString(name) {
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
	return "@p" + strconv.Itoa(paramIndex+1)
}

func (QueryFormatter) FormatStringLiteral(str string) string {
	return sqldb.FormatSingleQuoteStringLiteral(str)
}
