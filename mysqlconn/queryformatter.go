package mysqlconn

import (
	"fmt"
	"regexp"
	"strings"
)

// MySQL identifier rules:
// https://dev.mysql.com/doc/refman/8.0/en/identifiers.html
// Unquoted: letters, digits, $, _; may start with digit; max 64 chars.
// Schema-qualified: database.table
var (
	tableNameRegex  = regexp.MustCompile(`^([0-9a-zA-Z$_]{1,64}\.)?[0-9a-zA-Z$_]{1,64}$`)
	columnNameRegex = regexp.MustCompile(`^[0-9a-zA-Z$_]{1,64}$`)
)

// MySQL 8.0 reserved keywords.
// https://dev.mysql.com/doc/refman/8.0/en/reserved-words.html
var reservedWords = map[string]struct{}{
	"accessible": {}, "add": {}, "all": {}, "alter": {}, "analyze": {},
	"and": {}, "as": {}, "asc": {}, "asensitive": {}, "before": {},
	"between": {}, "bigint": {}, "binary": {}, "blob": {}, "both": {},
	"by": {}, "call": {}, "cascade": {}, "case": {}, "change": {},
	"char": {}, "character": {}, "check": {}, "collate": {}, "column": {},
	"condition": {}, "constraint": {}, "continue": {}, "convert": {},
	"create": {}, "cross": {}, "cume_dist": {}, "current_date": {},
	"current_time": {}, "current_timestamp": {}, "current_user": {},
	"cursor": {}, "database": {}, "databases": {}, "day_hour": {},
	"day_microsecond": {}, "day_minute": {}, "day_second": {}, "dec": {},
	"decimal": {}, "declare": {}, "default": {}, "delayed": {}, "delete": {},
	"dense_rank": {}, "desc": {}, "describe": {}, "deterministic": {},
	"distinct": {}, "distinctrow": {}, "div": {}, "double": {}, "drop": {},
	"dual": {}, "each": {}, "else": {}, "elseif": {}, "empty": {},
	"enclosed": {}, "escaped": {}, "except": {}, "exists": {}, "exit": {},
	"explain": {}, "false": {}, "fetch": {}, "first_value": {}, "float": {},
	"float4": {}, "float8": {}, "for": {}, "force": {}, "foreign": {},
	"from": {}, "fulltext": {}, "function": {}, "generated": {}, "get": {},
	"grant": {}, "group": {}, "grouping": {}, "groups": {}, "having": {},
	"high_priority": {}, "hour_microsecond": {}, "hour_minute": {},
	"hour_second": {}, "if": {}, "ignore": {}, "in": {}, "index": {},
	"infile": {}, "inner": {}, "inout": {}, "insensitive": {}, "insert": {},
	"int": {}, "int1": {}, "int2": {}, "int3": {}, "int4": {}, "int8": {},
	"integer": {}, "intersect": {}, "interval": {}, "into": {},
	"io_after_gtids": {}, "io_before_gtids": {}, "is": {}, "iterate": {},
	"join": {}, "json_table": {}, "key": {}, "keys": {}, "kill": {},
	"lag": {}, "last_value": {}, "lateral": {}, "lead": {}, "leading": {},
	"leave": {}, "left": {}, "like": {}, "limit": {}, "linear": {},
	"lines": {}, "load": {}, "localtime": {}, "localtimestamp": {},
	"lock": {}, "long": {}, "longblob": {}, "longtext": {}, "loop": {},
	"low_priority": {}, "master_bind": {}, "master_ssl_verify_server_cert": {},
	"match": {}, "maxvalue": {}, "mediumblob": {}, "mediumint": {},
	"mediumtext": {}, "middleint": {}, "minute_microsecond": {},
	"minute_second": {}, "mod": {}, "modifies": {}, "natural": {}, "not": {},
	"no_write_to_binlog": {}, "nth_value": {}, "ntile": {}, "null": {},
	"numeric": {}, "of": {}, "on": {}, "optimize": {}, "optimizer_costs": {},
	"option": {}, "optionally": {}, "or": {}, "order": {}, "out": {},
	"outer": {}, "outfile": {}, "over": {}, "partition": {},
	"percent_rank": {}, "precision": {}, "primary": {}, "procedure": {},
	"purge": {}, "range": {}, "rank": {}, "read": {}, "reads": {},
	"read_write": {}, "real": {}, "recursive": {}, "references": {},
	"regexp": {}, "release": {}, "rename": {}, "repeat": {}, "replace": {},
	"require": {}, "resignal": {}, "restrict": {}, "return": {}, "revoke": {},
	"right": {}, "rlike": {}, "row": {}, "rows": {}, "row_number": {},
	"schema": {}, "schemas": {}, "second_microsecond": {}, "select": {},
	"sensitive": {}, "separator": {}, "set": {}, "show": {}, "signal": {},
	"smallint": {}, "spatial": {}, "specific": {}, "sql": {},
	"sqlexception": {}, "sqlstate": {}, "sqlwarning": {},
	"sql_big_result": {}, "sql_calc_found_rows": {}, "sql_small_result": {},
	"ssl": {}, "starting": {}, "stored": {}, "straight_join": {},
	"system": {}, "table": {}, "terminated": {}, "then": {}, "tinyblob": {},
	"tinyint": {}, "tinytext": {}, "to": {}, "trailing": {}, "trigger": {},
	"true": {}, "undo": {}, "union": {}, "unique": {}, "unlock": {},
	"unsigned": {}, "update": {}, "usage": {}, "use": {}, "using": {},
	"utc_date": {}, "utc_time": {}, "utc_timestamp": {}, "values": {},
	"varbinary": {}, "varchar": {}, "varcharacter": {}, "varying": {},
	"virtual": {}, "when": {}, "where": {}, "while": {}, "window": {},
	"with": {}, "write": {}, "xor": {}, "year_month": {}, "zerofill": {},
}

// EscapeIdentifier wraps a MySQL identifier in backticks when necessary,
// escaping any backtick characters as “.
// Quoting is applied when the identifier contains non-lowercase/non-underscore
// characters or is a MySQL reserved word.
func EscapeIdentifier(ident string) string {
	escaped := strings.ReplaceAll(ident, "`", "``")
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
		return "`" + escaped + "`"
	}
	return ident
}

// QueryFormatter is the [sqldb.QueryFormatter] implementation for MySQL/MariaDB.
// Uses backtick identifier quoting, ? placeholders, and backslash+quote escaping for strings.
type QueryFormatter struct{}

// FormatTableName implements [sqldb.QueryFormatter.FormatTableName].
func (QueryFormatter) FormatTableName(name string) (string, error) {
	if !tableNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid MySQL table name %q", name)
	}
	if schema, table, ok := strings.Cut(name, "."); ok {
		return EscapeIdentifier(schema) + "." + EscapeIdentifier(table), nil
	}
	return EscapeIdentifier(name), nil
}

// FormatColumnName implements [sqldb.QueryFormatter.FormatColumnName].
func (QueryFormatter) FormatColumnName(name string) (string, error) {
	if !columnNameRegex.MatchString(name) {
		return "", fmt.Errorf("invalid MySQL column name %q", name)
	}
	return EscapeIdentifier(name), nil
}

// FormatPlaceholder implements [sqldb.QueryFormatter.FormatPlaceholder].
func (QueryFormatter) FormatPlaceholder(paramIndex int) string {
	if paramIndex < 0 {
		panic("paramIndex must be greater or equal zero")
	}
	return "?"
}

// FormatStringLiteral escapes a string for use as a MySQL string literal.
// Backslashes are escaped first (\→\\) then single quotes are doubled ('→”).
// Both steps are safe regardless of NO_BACKSLASH_ESCAPES mode.
func (QueryFormatter) FormatStringLiteral(str string) string {
	str = strings.ReplaceAll(str, `\`, `\\`)
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}

// MaxArgs implements [sqldb.QueryFormatter.MaxArgs].
func (QueryFormatter) MaxArgs() int {
	return 65535
}
