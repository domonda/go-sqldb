package impl

// Row is an interface with the methods of sql.Rows
// that are needed for ScanStruct.
// Allows mocking for tests without an SQL driver.
type Row interface {
	// Columns returns the column names.
	Columns() ([]string, error)
	// Scan copies the columns in the current row into the values pointed
	// at by dest. The number of values in dest must be the same as the
	// number of columns in Rows.
	Scan(dest ...any) error
}

type RowWithArrays interface {
	Row

	ScanWithArrays(dest []any) error
}

// func AsRowWithArrays(row Row) RowWithArrays {
// 	if r, ok := row.(RowWithArrays); ok {
// 		return r
// 	}
// 	return rowWithArrays{row}
// }

// type rowWithArrays struct {
// 	Row
// }

// func (r rowWithArrays) ScanWithArrays(dest []any) error {
// 	return ScanRowWithArrays(r.Row, dest)
// }
