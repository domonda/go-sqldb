package impl

import "github.com/domonda/go-sqldb"

// ScanStrings scans the values of a row as strings.
// Byte slices will be interpreted as strings,
// nil (SQL NULL) will be converted to an empty string,
// all other types are converted with fmt.Sprint(src).
func ScanStrings(src Row) ([]string, error) {
	cols, err := src.Columns()
	if err != nil {
		return nil, err
	}
	s := make([]string, len(cols))
	p := make([]interface{}, len(cols))
	for i := range p {
		p[i] = (*sqldb.StringScannable)(&s[i])
	}
	err = src.Scan(p...)
	if err != nil {
		return nil, err
	}
	return s, nil
}
