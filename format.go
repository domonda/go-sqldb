package sqldb

import (
	"database/sql/driver"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/lib/pq"
)

const timeFormat = "'2006-01-02 15:04:05.999999Z07:00:00'"

// type StringFormatter interface {
// 	StringLiteral(string) string
// }

// type StringFormatterFunc func(string) string

// func (f StringFormatterFunc) StringLiteral(s string) string {
// 	return f(s)
// }

type QueryFormatter interface {
	StringLiteral(str string) string
	ArrayLiteral(array any) (string, error)
	ValidateColumnName(name string) error
	ParameterPlaceholder(index int) string
	MaxParameters() int
}

type defaultQueryFormatter struct{}

func (defaultQueryFormatter) StringLiteral(str string) string {
	return defaultStringLiteral(str)
}

func (defaultQueryFormatter) ArrayLiteral(array any) (string, error) {
	value, err := pq.Array(array).Value()
	if err != nil {
		return "", fmt.Errorf("can't format %T as SQL array because: %w", array, err)
	}
	return fmt.Sprintf("'%s'", value), nil
}

func (defaultQueryFormatter) ValidateColumnName(name string) error {
	if name == `` || name == `""` {
		return errors.New("empty column name")
	}
	if strings.ContainsFunc(name, unicode.IsSpace) {
		return fmt.Errorf("column name %q contains whitespace", name)
	}
	if strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Errorf("column name %q contains control characters", name)
	}
	return nil
}

func (defaultQueryFormatter) ParameterPlaceholder(index int) string {
	return fmt.Sprintf("$%d", index+1)
}

func (defaultQueryFormatter) MaxParameters() int { return 1024 }

// AlwaysFormatValue formats a value for debugging or logging SQL statements.
// In case of any problems fmt.Sprint(val) is returned.
func AlwaysFormatValue(val any, formatter QueryFormatter) string {
	str, err := FormatValue(val, formatter)
	if err != nil {
		return fmt.Sprint(val)
	}
	return str
}

// FormatValue formats a value for debugging or logging SQL statements.
func FormatValue(val any, formatter QueryFormatter) (string, error) {
	if val == nil {
		return "NULL", nil
	}

	v := reflect.ValueOf(val)

	switch x := val.(type) {
	case driver.Valuer:
		if v.Kind() == reflect.Ptr && v.IsNil() {
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
		return FormatValue(value, formatter)

	case time.Time:
		return x.Format(timeFormat), nil
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return "NULL", nil
		}
		return FormatValue(v.Elem().Interface(), formatter)

	case reflect.Bool:
		if v.Bool() {
			return "TRUE", nil
		} else {
			return "FALSE", nil
		}

	case reflect.String:
		s := v.String()
		if l := len(s); l >= 2 && (s[0] == '{' && s[l-1] == '}' || s[0] == '[' && s[l-1] == ']') {
			// String is already an array literal, just quote it
			return `'` + s + `'`, nil
		}
		return formatter.StringLiteral(s), nil

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b := v.Bytes()
			if !utf8.Valid(b) {
				return `'\x` + hex.EncodeToString(b) + "'", nil
			}
			if l := len(b); l >= 2 && (b[0] == '{' && b[l-1] == '}' || b[0] == '[' && b[l-1] == ']') {
				return `'` + string(b) + `'`, nil
			}
			return formatter.StringLiteral(string(b)), nil
		}
		return formatter.ArrayLiteral(v.Interface())

	case reflect.Array:
		return formatter.ArrayLiteral(v.Interface())
	}

	return fmt.Sprint(val), nil
}

func FormatQuery(query string, args []any, naming QueryFormatter) string {
	// Replace placeholders with formatted args
	for i := len(args) - 1; i >= 0; i-- {
		placeholder := naming.ParameterPlaceholder(i)
		formattedArg := AlwaysFormatValue(args[i], naming)
		query = strings.ReplaceAll(query, placeholder, formattedArg)
	}

	// Line endings and indentation:

	lines := strings.Split(query, "\n")
	if len(lines) == 1 {
		return strings.TrimSpace(query)
	}

	// Trim whitespace at end of line and remove empty lines
	for i := 0; i < len(lines); i++ {
		lines[i] = strings.TrimRightFunc(lines[i], unicode.IsSpace)
		if lines[i] == "" {
			lines = append(lines[:i], lines[i+1:]...)
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

func defaultStringLiteral(literal string) string {
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
