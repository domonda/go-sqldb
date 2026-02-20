package sqldb

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const timeFormat = "'2006-01-02 15:04:05.999999Z07:00'"

// FormatValue formats a value for debugging or logging SQL statements.
func FormatValue(val any) (string, error) {
	if val == nil {
		return "NULL", nil
	}

	v := reflect.ValueOf(val)

	switch x := val.(type) {
	case driver.Valuer:
		if v.Kind() == reflect.Pointer && v.IsNil() {
			// Assume nil pointer implementing driver.Valuer is NULL
			// because if the method Value is implemented by value
			// the nil pointer will still implement driver.Valuer
			// but calling Value by value on a nil pointer panics
			return "NULL", nil
		}
		value, err := x.Value()
		if err != nil {
			return "", err
		}
		return FormatValue(value)

	case time.Time:
		return x.Format(timeFormat), nil
	}

	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return "NULL", nil
		}
		return FormatValue(v.Elem().Interface())

	case reflect.Bool:
		if v.Bool() {
			return "TRUE", nil
		} else {
			return "FALSE", nil
		}

	case reflect.String:
		s := v.String()
		if l := len(s); l >= 2 && (s[0] == '{' && s[l-1] == '}' || s[0] == '[' && s[l-1] == ']') {
			return `'` + s + `'`, nil
		}
		return QuoteLiteral(s), nil

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b := v.Bytes()
			if !utf8.Valid(b) {
				return `'\x` + hex.EncodeToString(b) + "'", nil
			}
			if l := len(b); l >= 2 && (b[0] == '{' && b[l-1] == '}' || b[0] == '[' && b[l-1] == ']') {
				return `'` + string(b) + `'`, nil
			}
			return QuoteLiteral(string(b)), nil
		}
	}

	return fmt.Sprint(val), nil
}

func NormalizeAndFormatQuery(normalize NormalizeQueryFunc, formatter QueryFormatter, query string, args ...any) (string, error) {
	if normalize == nil {
		normalize = NoChangeNormalizeQuery
	}
	if formatter == nil {
		formatter = NewQueryFormatter("$")
	}
	query, err := normalize(query)
	if err != nil {
		return "", err
	}
	return FormatQuery(formatter, query, args...), nil
}

func MustNormalizeAndFormatQuery(normalize NormalizeQueryFunc, formatter QueryFormatter, query string, args ...any) string {
	query, err := NormalizeAndFormatQuery(normalize, formatter, query, args...)
	if err != nil {
		panic("NormalizeAndFormatQuery error: " + err.Error())
	}
	return query
}

func FormatQuery(f QueryFormatter, query string, args ...any) string {
	if len(args) > 0 {
		if placeholder := f.FormatPlaceholder(0); f.FormatPlaceholder(1) == placeholder {
			// Uniform placeholders, replace every instance with one arg
			for _, arg := range args {
				value, err := FormatValue(arg)
				if err != nil {
					value = "FORMATERROR:" + err.Error()
				}
				// Note that this will replace placeholders in comments and strings
				query = strings.Replace(query, placeholder, value, 1)
			}
		} else {
			// Numbered placeholders, replace in reverse order
			// to avoid replacing shorter placeholders contained in longer ones
			for i := len(args) - 1; i >= 0; i-- {
				placeholder := f.FormatPlaceholder(i)
				value, err := FormatValue(args[i])
				if err != nil {
					value = "FORMATERROR:" + err.Error()
				}
				// Note that this will replace placeholders in comments and strings
				query = strings.ReplaceAll(query, placeholder, value)
			}
		}
	}

	lines := strings.Split(query, "\n")
	if len(lines) == 1 {
		return strings.TrimSpace(query)
	}

	// Trim whitespace at end of line and remove empty lines
	for i := 0; i < len(lines); i++ {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		if lines[i] == "" {
			lines = slices.Delete(lines, i, i+1)
			i--
		}
	}

	// Remove identical whitespace at beginning of each line
	firstLineRune, runeSize := utf8.DecodeRuneInString(lines[0])
	for unicode.IsSpace(firstLineRune) {
		identical := true
		for i := 1; i < len(lines); i++ {
			lineRune, _ := utf8.DecodeRuneInString(lines[i])
			if lineRune != firstLineRune {
				identical = false
				break
			}
		}
		if !identical {
			break
		}
		for i := range lines {
			lines[i] = lines[i][runeSize:]
		}
		firstLineRune, _ = utf8.DecodeRuneInString(lines[0])
	}

	return strings.Join(lines, "\n")
}

// QuoteLiteral quotes a 'literal' (e.g. a parameter, often used to pass literal
// to DDL and other statements that do not accept parameters) to be used as part
// of an SQL statement.  For example:
//
//	exp_date := pq.QuoteLiteral("2023-01-05 15:00:00Z")
//	err := db.Exec(fmt.Sprintf("CREATE ROLE my_user VALID UNTIL %s", exp_date))
//
// Any single quotes in name will be escaped. Any backslashes (i.e. "\") will be
// replaced by two backslashes (i.e. "\\") and the C-style escape identifier
// that PostgreSQL provides ('E') will be prepended to the string.
func QuoteLiteral(literal string) string {
	// This follows the PostgreSQL internal algorithm for handling quoted literals
	// from libpq, which can be found in the "PQEscapeStringInternal" function,
	// which is found in the libpq/fe-exec.c source file:
	// https://git.postgresql.org/gitweb/?p=postgresql.git;a=blob;f=src/interfaces/libpq/fe-exec.c
	//
	// substitute any single-quotes (') with two single-quotes ('')
	literal = strings.Replace(literal, `'`, `''`, -1)
	// determine if the string has any backslashes (\) in it.
	// if it does, replace any backslashes (\) with two backslashes (\\)
	// then, we need to wrap the entire string with a PostgreSQL
	// C-style escape. Per how "PQEscapeStringInternal" handles this case, we
	// also add a space before the "E"
	if strings.Contains(literal, `\`) {
		literal = strings.Replace(literal, `\`, `\\`, -1)
		literal = ` E'` + literal + `'`
	} else {
		// otherwise, we can just wrap the literal with a pair of single quotes
		literal = `'` + literal + `'`
	}
	return literal
}
