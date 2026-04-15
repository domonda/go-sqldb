package db

import (
	// Imported so doc-link references like [driver.Value] and [time.Time]
	// in the godoc of [ScanConverter], [BytesToStringScanConverter], and
	// [TimeToStringScanConverter] resolve to clickable links.
	_ "database/sql/driver"
	_ "time"

	"github.com/domonda/go-sqldb"
)

// ScanConverter converts a [driver.Value] into another value
// before it is scanned into a destination.
// The second return value reports whether the conversion was applied;
// if false the original value is used unchanged.
type ScanConverter = sqldb.ScanConverter

// ScanConverterFunc adapts a function to the [ScanConverter] interface.
type ScanConverterFunc = sqldb.ScanConverterFunc

// ScanConverters is a slice of [ScanConverter] that itself implements
// [ScanConverter] by trying each converter in order and returning
// the result of the first one that reports a successful conversion.
type ScanConverters = sqldb.ScanConverters

// ScanConvertValueOrUnchanged returns the result of the first converter
// that reports a successful conversion of value,
// or value unchanged if no converter applied.
func ScanConvertValueOrUnchanged(value any, converters ...ScanConverter) any {
	return sqldb.ScanConvertValueOrUnchanged(value, converters...)
}

// BytesToStringScanConverter returns a [ScanConverterFunc] that converts
// [driver.Value] of type []byte to a string.
// If the bytes are valid UTF-8 they are returned as a string unchanged.
// Otherwise they are encoded as uppercase hex and prefixed with hexPrefix,
// producing strings like "\xDEADBEEF" or "0xDEADBEEF" depending on the prefix.
// Values that are not of type []byte are returned as (nil, false)
// so the converter can be chained with others.
//
// Common hexPrefix values are `\x` (PostgreSQL bytea hex format)
// or `0x` (Go and SQL hex literal).
func BytesToStringScanConverter(hexPrefix string) ScanConverterFunc {
	return sqldb.BytesToStringScanConverter(hexPrefix)
}

// TimeToStringScanConverter returns a [ScanConverterFunc] that formats
// [driver.Value] of type [time.Time] as a string using the given layout
// (see [time.Time.Format]).
// Values that are not of type [time.Time] are returned as (nil, false)
// so the converter can be chained with others.
//
// Common layouts are [time.RFC3339], [time.RFC3339Nano], or [time.DateTime].
func TimeToStringScanConverter(layout string) ScanConverterFunc {
	return sqldb.TimeToStringScanConverter(layout)
}
