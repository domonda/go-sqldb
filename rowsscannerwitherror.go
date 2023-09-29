package sqldb

// RowsScannerWithError returns a dummy RowsScanner
// where all methods return the passed error.
func RowsScannerWithError(err error) RowsScanner {
	return rowsScannerWithError{err}
}

type rowsScannerWithError struct {
	err error
}

func (e rowsScannerWithError) ScanSlice(dest any) error {
	return e.err
}

func (e rowsScannerWithError) ScanStructSlice(dest any) error {
	return e.err
}

func (e rowsScannerWithError) Columns() ([]string, error) {
	return nil, e.err
}

func (e rowsScannerWithError) ScanAllRowsAsStrings(headerRow bool) ([][]string, error) {
	return nil, e.err
}

func (e rowsScannerWithError) ForEachRow(callback func(RowScanner) error) error {
	return e.err
}

func (e rowsScannerWithError) ForEachRowCall(callback any) error {
	return e.err
}
