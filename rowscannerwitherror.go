package sqldb

// RowScannerWithError returns a dummy RowScanner
// where all methods return the passed error.
func RowScannerWithError(err error) RowScanner {
	return rowScannerWithError{err}
}

type rowScannerWithError struct {
	err error
}

func (e rowScannerWithError) Scan(dest ...any) error {
	return e.err
}

func (e rowScannerWithError) ScanStruct(dest any) error {
	return e.err
}

func (e rowScannerWithError) ScanValues() ([]any, error) {
	return nil, e.err
}

func (e rowScannerWithError) ScanStrings() ([]string, error) {
	return nil, e.err
}

func (e rowScannerWithError) Columns() ([]string, error) {
	return nil, e.err
}
