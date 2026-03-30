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
		return QuoteStringLiteral(s), nil

	case reflect.Slice:
		if v.Type().Elem().Kind() == reflect.Uint8 {
			b := v.Bytes()
			if !utf8.Valid(b) {
				return `'\x` + hex.EncodeToString(b) + "'", nil
			}
			if l := len(b); l >= 2 && (b[0] == '{' && b[l-1] == '}' || b[0] == '[' && b[l-1] == ']') {
				return `'` + string(b) + `'`, nil
			}
			return QuoteStringLiteral(string(b)), nil
		}
	}

	return fmt.Sprint(val), nil
}

// NormalizeAndFormatQuery normalizes the query using the given function, then
// substitutes argument placeholders using the formatter for display purposes.
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

// MustNormalizeAndFormatQuery is like NormalizeAndFormatQuery but panics on error.
func MustNormalizeAndFormatQuery(normalize NormalizeQueryFunc, formatter QueryFormatter, query string, args ...any) string {
	query, err := NormalizeAndFormatQuery(normalize, formatter, query, args...)
	if err != nil {
		panic("NormalizeAndFormatQuery error: " + err.Error())
	}
	return query
}

// FormatQuery substitutes argument placeholders in the query with formatted values
// using the given QueryFormatter, returning a human-readable SQL string.
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
		firstLineRune, runeSize = utf8.DecodeRuneInString(lines[0])
	}

	return strings.Join(lines, "\n")
}

// QuoteStringLiteral formats a string as an ANSI SQL single-quoted literal,
// doubling any embedded single quotes.
func QuoteStringLiteral(str string) string {
	return "'" + strings.ReplaceAll(str, "'", "''") + "'"
}
