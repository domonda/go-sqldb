package sqldb

import (
	"database/sql/driver"
	"encoding/hex"
	"errors"
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
// A value implementing the [Secret] interface is formatted as its redacted
// String representation so the wrapped value is never revealed.
func FormatValue(val any) (string, error) {
	if val == nil {
		return "NULL", nil
	}

	v := reflect.ValueOf(val)

	switch x := val.(type) {
	case Secret:
		// Never reveal the wrapped value, use the redacted String instead
		return QuoteStringLiteral(x.String()), nil

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
//
// Arguments wrapped with [KeepSecret] appear redacted in the result.
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

// formatErrorSentinel is substituted for a placeholder whose argument
// could not be formatted by [FormatValue]. It keeps the positional
// alignment of subsequent placeholders correct in the uniform-placeholder
// case (where replacements consume placeholders one at a time) and makes
// the failed slot visible in the formatted output. The collected error
// is still returned alongside the substituted query.
const formatErrorSentinel = "<FORMATERROR>"

// SubstitutePlaceholders substitutes argument placeholders in the query with
// formatted values using [FormatValue] and the given QueryFormatter's
// FormatPlaceholder method. If formatting an arg fails, the placeholder is
// replaced with the sentinel "<FORMATERROR>" so subsequent uniform
// placeholders stay aligned with their arguments, and the error is collected;
// all collected errors are returned joined via [errors.Join] together with
// the substituted query string.
//
// Arguments wrapped with [KeepSecret] are substituted as a redacted
// placeholder instead of their actual value.
func SubstitutePlaceholders(formatter QueryFormatter, query string, args []any) (string, error) {
	if len(args) == 0 {
		return query, nil
	}
	var errs []error
	if placeholder := formatter.FormatPlaceholder(0); formatter.FormatPlaceholder(1) == placeholder {
		// Uniform placeholders, replace every instance with one arg
		for _, arg := range args {
			value, err := FormatValue(arg)
			if err != nil {
				errs = append(errs, err)
				value = formatErrorSentinel
			}
			// Note that this will replace placeholders in comments and strings
			query = strings.Replace(query, placeholder, value, 1)
		}
	} else {
		// Numbered placeholders, replace in reverse order
		// to avoid replacing shorter placeholders contained in longer ones
		for i := len(args) - 1; i >= 0; i-- {
			placeholder := formatter.FormatPlaceholder(i)
			value, err := FormatValue(args[i])
			if err != nil {
				errs = append(errs, err)
				value = formatErrorSentinel
			}
			// Note that this will replace placeholders in comments and strings
			query = strings.ReplaceAll(query, placeholder, value)
		}
	}
	return query, errors.Join(errs...)
}

// FormatQuery trims surrounding whitespace from the query and substitutes
// argument placeholders with formatted values using the given QueryFormatter,
// returning a human-readable SQL string that is meant for debugging and logging,
// not for execution.
//
// If the formatter returns an error, the error is appended to the trimmed query
// to not silently discard errors.
//
// Arguments wrapped with [KeepSecret] appear redacted in the result.
func FormatQuery(formatter QueryFormatter, query string, args ...any) string {
	trimmed := TrimSurroundingWhitespace(query)
	if formatter == nil {
		return trimmed
	}
	substituted, err := formatter.SubstitutePlaceholders(trimmed, args)
	if err != nil {
		// Append placeholder substitution error to the trimmed query
		// to not silently discard errors.
		// The result is for debugging and logging, not for execution.
		return trimmed + "\n\nPlaceholder substitution error: " + err.Error()
	}
	return substituted
}

// TrimSurroundingWhitespace trims surrounding whitespace from a SQL query string.
// Single-line queries are trimmed on both ends. Multi-line queries have
// trailing whitespace and empty lines removed, and any whitespace prefix
// common to every line is stripped, so the result keeps its relative
// indentation without the leading margin.
func TrimSurroundingWhitespace(query string) string {
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

	if len(lines) == 0 {
		return ""
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
