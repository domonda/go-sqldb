package impl

// Row is an interface with the methods of sql.Rows
// that are needed for ScanSlice.
// Allows mocking for tests without an SQL driver.
type Rows interface {
	Row

	// Close closes the Rows, preventing further enumeration. If Next is called
	// and returns false and there are no further result sets,
	// the Rows are closed automatically and it will suffice to check the
	// result of Err. Close is idempotent and does not affect the result of Err.
	Close() error

	// Next prepares the next result row for reading with the Scan method. It
	// returns true on success, or false if there is no next result row or an error
	// happened while preparing it. Err should be consulted to distinguish between
	// the two cases.
	//
	// Every call to Scan, even the first one, must be preceded by a call to Next.
	Next() bool

	// Err returns the error, if any, that was encountered during iteration.
	// Err may be called after an explicit or implicit Close.
	Err() error
}

// RowAsRows implements the methods of Rows for a Row as no-ops.
// Note that Next() always returns true leading to an endless loop
// if used to scan multiple rows.
func RowAsRows(row Row) Rows {
	return rowAsRows{Row: row}
}

type rowAsRows struct {
	Row
}

func (rowAsRows) Close() error { return nil }
func (rowAsRows) Next() bool   { return true }
func (rowAsRows) Err() error   { return nil }
