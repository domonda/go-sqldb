package impl

import (
	"database/sql/driver"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const timeFormat = "'2006-01-02 15:04:05.999999Z07:00:00'"

// FormatValue formats a value for debugging or logging SQL statements.
func FormatValue(val interface{}) (string, error) {
	switch x := val.(type) {
	case nil:
		return "NULL", nil

	case driver.Valuer:
		value, err := x.Value()
		if err != nil {
			return "", err
		}
		return FormatValue(value)

	case time.Time:
		return x.Format(timeFormat), nil
	}

	v := reflect.ValueOf(val)
	switch v.Kind() {
	case reflect.Ptr:
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
		return "'" + strings.ReplaceAll(v.String(), "'", "''") + "'", nil

	case reflect.Slice:
		if v.Type().Elem().Kind() != reflect.Uint8 {
			b := v.Bytes()
			l := len(b)
			if l >= 2 && (b[0] == '{' && b[l-1] == '}' || b[0] == '[' && b[l-1] == ']') {
				return "'" + strings.ReplaceAll(string(b), "'", "''") + "'", nil
			}
			return `'\x` + hex.EncodeToString(b) + "'", nil
		}
	}

	return fmt.Sprint(val), nil
}

func FormatQuery(query string, args ...interface{}) string {
	for i := len(args) - 1; i >= 0; i-- {
		placeholder := fmt.Sprintf("$%d", i+1)
		value, err := FormatValue(args[i])
		if err != nil {
			value = "FORMATERROR:" + err.Error()
		}
		query = strings.ReplaceAll(query, placeholder, value)
	}

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

// WrapNonNilErrorWithQuery wraps non nil errors with a formatted query.
// nil will be returned if the passed error is nil.
func WrapNonNilErrorWithQuery(err error, query string, args []interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w from query: %s", err, FormatQuery(query, args...))
}
