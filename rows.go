package sqldb

// Rows is an interface with the methods of sql.Rows.
// Allows mocking for tests without an SQL driver.
type Rows interface {
	// Columns returns the column names.
	Columns() ([]string, error)

	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...any) error

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

func RowsWithError(err error) Rows {
	return rowsWithError{err}
}

type rowsWithError struct {
	err error
}

func (r rowsWithError) Columns() ([]string, error) { return nil, r.err }
func (r rowsWithError) Scan(dest ...any) error     { return r.err }
func (r rowsWithError) Close() error               { return nil }
func (r rowsWithError) Next() bool                 { return false }
func (r rowsWithError) Err() error                 { return r.err }
