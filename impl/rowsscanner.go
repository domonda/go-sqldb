package impl

import (
	"context"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// RowsScanner implements sqldb.RowsScanner with Rows
type RowsScanner struct {
	Context          context.Context
	Query            string // for error wrapping
	Rows             Rows
	StructFieldNamer sqldb.StructFieldNamer
}

func (s *RowsScanner) ScanSlice(dest interface{}) error {
	err := ScanRowsAsSlice(s.Context, s.Rows, dest, nil)
	if err != nil {
		return fmt.Errorf("query `%s` returned error: %w", s.Query, err)
	}
	return nil
}

func (s *RowsScanner) ScanStructSlice(dest interface{}) error {
	err := ScanRowsAsSlice(s.Context, s.Rows, dest, s.StructFieldNamer)
	if err != nil {
		return fmt.Errorf("query `%s` returned error: %w", s.Query, err)
	}
	return nil
}

func (s *RowsScanner) ScanStrings() (rows [][]string, err error) {
	err = s.ForEachRow(func(rowScanner sqldb.RowScanner) error {
		row, err := rowScanner.ScanStrings()
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
		if err != nil {
			err = fmt.Errorf("query `%s` returned error: %w", s.Query, err)
		}
		s.Rows.Close()
	}()

	for s.Rows.Next() {
		if s.Context.Err() != nil {
			return s.Context.Err()
		}

		err := callback(CurrentRowScanner{s.Rows, s.StructFieldNamer})
		if err != nil {
			return err
		}
	}
	return s.Rows.Err()
}

func (s *RowsScanner) ForEachRowScan(callback interface{}) error {
	forEachRowFunc, err := ForEachRowScanFunc(s.Context, callback)
	if err != nil {
		return err
	}
	return s.ForEachRow(forEachRowFunc)
}
