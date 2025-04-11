package sqldb

// import (
// 	"database/sql"
// 	"errors"
// )

// // Row is an interface with methods from sql.Rows
// // that are needed for ScanStruct.
// type Row interface {
// 	// SingleRow is a marker method to distinguish from sql.Rows
// 	SingleRow()

// 	// Columns returns the column names.
// 	Columns() ([]string, error)
// 	// Scan copies the columns in the current row into the values pointed
// 	// at by dest. The number of values in dest must be the same as the
// 	// number of columns in Rows.
// 	Scan(dest ...any) error
// }

// func RowFromRows(rows Rows) Row {
// 	return &singleRow{rows}
// }

// type singleRow struct {
// 	Rows
// }

// func (*singleRow) SingleRow() {}

// func (s *singleRow) Scan(dest ...any) (err error) {
// 	defer func() {
// 		err = errors.Join(err, s.Rows.Close())
// 	}()

// 	if len(dest) == 0 {
// 		return errors.New("Scan called with no destination arguments")
// 	}
// 	// Check if there was an error even before preparing the row with Next()
// 	if s.Rows.Err() != nil {
// 		return s.Rows.Err()
// 	}
// 	if !s.Rows.Next() {
// 		// Error during preparing the row with Next()
// 		if s.Rows.Err() != nil {
// 			return s.Rows.Err()
// 		}
// 		return sql.ErrNoRows
// 	}

// 	return s.Rows.Scan(dest...)
// }
