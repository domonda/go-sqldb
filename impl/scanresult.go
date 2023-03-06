package impl

import "github.com/domonda/go-sqldb"

// ScanValues returns the values of a row exactly how they are
// passed from the database driver to an sql.Scanner.
// Byte slices will be copied.
func ScanValues(src Row) ([]any, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		anys   = make([]sqldb.AnyValue, len(cols))
		result = make([]any, len(cols))
	)
	// result elements hold pointer to sqldb.AnyValue for scanning
	for i := range result {
		result[i] = &anys[i]
	}
	err = src.Scan(result...)
	if err != nil {
		return nil, err
	}
	// don't return pointers to sqldb.AnyValue
	// but what internal value has been scanned
	for i := range result {
		result[i] = anys[i].Val
	}
	return result, nil
}

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint.
func ScanStrings(src Row) ([]string, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	var (
		result     = make([]string, len(cols))
		resultPtrs = make([]any, len(cols))
	)
	for i := range resultPtrs {
		resultPtrs[i] = (*sqldb.StringScannable)(&result[i])
	}
	err = src.Scan(resultPtrs...)
	if err != nil {
		return nil, err
	}
	return result, nil
}
