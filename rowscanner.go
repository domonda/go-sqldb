package sqldb

// RowScanner scans the values from a single row.
type RowScanner interface {
	// Scan values of a row into dest variables, which must be passed as pointers.
	Scan(dest ...any) error

	// ScanStruct scans values of a row into a dest struct which must be passed as pointer.
	ScanStruct(dest any) error

	// ScanValues returns the values of a row exactly how they are
	// passed from the database driver to an sql.Scanner.
	// Byte slices will be copied.
	ScanValues() ([]any, error)

	// ScanStrings scans the values of a row as strings.
	// Byte slices will be interpreted as strings,
	// nil (SQL NULL) will be converted to an empty string,
	// all other types are converted with fmt.Sprint(src).
	ScanStrings() ([]string, error)
}
