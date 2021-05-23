package impl

import (
	"context"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// RowsScanner implements sqldb.RowsScanner with Rows
type RowsScanner struct {
	ctx              context.Context // ctx is checked for every row and passed through to callbacks
	rows             Rows
	structFieldNamer sqldb.StructFieldNamer
	query            string        // for error wrapping
	argFmt           string        // for error wrapping
	args             []interface{} // for error wrapping
}

func NewRowsScanner(ctx context.Context, rows Rows, structFieldNamer sqldb.StructFieldNamer, query, argFmt string, args []interface{}) *RowsScanner {
	return &RowsScanner{ctx, rows, structFieldNamer, query, argFmt, args}
}

func (s *RowsScanner) ScanSlice(dest interface{}) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, nil)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *RowsScanner) ScanStructSlice(dest interface{}) error {
	err := ScanRowsAsSlice(s.ctx, s.rows, dest, s.structFieldNamer)
	if err != nil {
		return fmt.Errorf("%w from query: %s", err, FormatQuery(s.query, s.argFmt, s.args...))
	}
	return nil
}

func (s *RowsScanner) ScanStrings(headerRow bool) (rows [][]string, err error) {
	cols, err := s.rows.Columns()
	if err != nil {
		return nil, err
	}
	if headerRow {
		rows = [][]string{cols}
	}
	stringScannablePtrs := make([]interface{}, len(cols))
	err = s.ForEachRow(func(rowScanner sqldb.RowScanner) error {
		row := make([]string, len(cols))
		for i := range stringScannablePtrs {
			stringScannablePtrs[i] = (*sqldb.StringScannable)(&row[i])
		}
		err := rowScanner.Scan(stringScannablePtrs...)
		if err != nil {
			return err
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *RowsScanner) ForEachRow(callback func(sqldb.RowScanner) error) (err error) {
	defer func() {
		s.rows.Close()
		err = WrapNonNilErrorWithQuery(err, s.query, s.argFmt, s.args)
	}()

	for s.rows.Next() {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}

		err := callback(CurrentRowScanner{s.rows, s.structFieldNamer})
		if err != nil {
			return err
		}
	}
	return s.rows.Err()
}

func (s *RowsScanner) ForEachRowCall(callback interface{}) error {
	forEachRowFunc, err := ForEachRowCallFunc(s.ctx, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}
