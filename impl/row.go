package impl

import "reflect"

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

func ScanRowWithArrays(row Row, dest []any) error {
	if r, ok := row.(RowWithArrays); ok {
		return r.ScanWithArrays(dest)
	}
	var wrappedDest []any
	for i, d := range dest {
		if ShouldWrapForArrayScanning(reflect.ValueOf(d).Elem()) {
			if wrappedDest == nil {
				// Allocate new slice for wrapped element
				wrappedDest = make([]any, len(dest))
				// Copy previous elements
				for h := 0; h < i; h++ {
					wrappedDest[h] = dest[h]
				}
			}
			wrappedDest[i] = WrapForArrayScanning(d)
		} else if wrappedDest != nil {
			wrappedDest[i] = d
		}
	}
	if wrappedDest != nil {
		return row.Scan(wrappedDest...)
	}
	return row.Scan(dest...)
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
