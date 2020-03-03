package sqldb

// RowScanner scans the values from a single row.
type RowScanner interface {
	// Scan values of a row into dest variables, which must be passed as pointers.
	Scan(dest ...interface{}) error

	// Scan values of a row into a dest struct which must be passed as pointer.
	ScanStruct(dest interface{}) error
}
