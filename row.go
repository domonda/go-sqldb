package sqldb

// Row is an interface with methods from sql.Rows
// that are needed for ScanStruct.
type Row interface {
	// Columns returns the column names.
	Columns() ([]string, error)
	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...any) error
}