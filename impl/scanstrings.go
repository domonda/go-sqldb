package impl

import "github.com/domonda/go-sqldb"

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint.
func ScanStrings(src Row) ([]string, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	row := make([]string, len(cols))
	args := make([]interface{}, len(cols))
	for i := range args {
		args[i] = (*sqldb.StringScannable)(&row[i])
	}
	err = src.Scan(args...)
	if err != nil {
		return nil, err
	}
	return row, nil
}
