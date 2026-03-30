package sqliteconn

import (
	"database/sql"
	"fmt"
	"math"

	"zombiezen.com/go/sqlite"
)

type rows struct {
	stmt               *sqlite.Stmt
	conn               *sqlite.Conn
	hasRow             bool
	err                error
	closed             bool
	shouldFinalizeStmt bool // Whether this rows object should finalize the statement
}

func (r *rows) Next() bool {
	if r.closed || r.err != nil {
		return false
	}

	hasRow, err := r.stmt.Step()
	if err != nil {
		r.err = wrapKnownErrors(err)
		return false
	}

	r.hasRow = hasRow
	return hasRow
}

func (r *rows) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if !r.hasRow {
		return sql.ErrNoRows
	}

	columnCount := r.stmt.ColumnCount()
	if len(dest) != columnCount {
		return fmt.Errorf("expected %d destination arguments in Scan, got %d", columnCount, len(dest))
	}

	for i := range dest {
		if err := scanColumn(r.stmt, i, dest[i]); err != nil {
			return err
		}
	}

	return nil
}

func (r *rows) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true

	// Only finalize the statement if this rows object owns it
	if r.shouldFinalizeStmt {
		return r.stmt.Finalize()
	}

	// Otherwise just reset the statement for reuse
	return r.stmt.Reset()
}

func (r *rows) Err() error {
	return r.err
}

func (r *rows) Columns() ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}

	columnCount := r.stmt.ColumnCount()
	columns := make([]string, columnCount)
	for i := range columnCount {
		columns[i] = r.stmt.ColumnName(i)
	}
	return columns, nil
}

// scanColumn scans a single column value into the destination
func scanColumn(stmt *sqlite.Stmt, col int, dest any) error {
	colType := stmt.ColumnType(col)

	switch colType {
	case sqlite.TypeNull:
		// Handle NULL values
		switch d := dest.(type) {
		case *any:
			*d = nil
		case *sql.NullString:
			d.Valid = false
		case *sql.NullInt64:
			d.Valid = false
		case *sql.NullFloat64:
			d.Valid = false
		case *sql.NullBool:
			d.Valid = false
		case *sql.NullInt32:
			d.Valid = false
		case *sql.NullInt16:
			d.Valid = false
		case *sql.NullByte:
			d.Valid = false
		case *sql.NullTime:
			d.Valid = false
		case **string:
			*d = nil
		case **int:
			*d = nil
		case **int64:
			*d = nil
		case **float64:
			*d = nil
		case **bool:
			*d = nil
		case *[]byte:
			*d = nil
		default:
			if scanner, ok := dest.(sql.Scanner); ok {
				return scanner.Scan(nil)
			}
			return fmt.Errorf("cannot scan NULL into %T", dest)
		}
		return nil

	case sqlite.TypeInteger:
		val := stmt.ColumnInt64(col)
		switch d := dest.(type) {
		case *int:
			if val < math.MinInt || val > math.MaxInt {
				return fmt.Errorf("integer overflow: %d does not fit in int", val)
			}
			*d = int(val)
		case *int8:
			if val < math.MinInt8 || val > math.MaxInt8 {
				return fmt.Errorf("integer overflow: %d does not fit in int8", val)
			}
			*d = int8(val)
		case *int16:
			if val < math.MinInt16 || val > math.MaxInt16 {
				return fmt.Errorf("integer overflow: %d does not fit in int16", val)
			}
			*d = int16(val)
		case *int32:
			if val < math.MinInt32 || val > math.MaxInt32 {
				return fmt.Errorf("integer overflow: %d does not fit in int32", val)
			}
			*d = int32(val)
		case *int64:
			*d = val
		case *uint:
			if val < 0 {
				return fmt.Errorf("integer overflow: %d does not fit in uint", val)
			}
			*d = uint(val)
		case *uint8:
			if val < 0 || val > math.MaxUint8 {
				return fmt.Errorf("integer overflow: %d does not fit in uint8", val)
			}
			*d = uint8(val)
		case *uint16:
			if val < 0 || val > math.MaxUint16 {
				return fmt.Errorf("integer overflow: %d does not fit in uint16", val)
			}
			*d = uint16(val)
		case *uint32:
			if val < 0 || val > math.MaxUint32 {
				return fmt.Errorf("integer overflow: %d does not fit in uint32", val)
			}
			*d = uint32(val)
		case *uint64:
			if val < 0 {
				return fmt.Errorf("integer overflow: %d does not fit in uint64", val)
			}
			*d = uint64(val)
		case *bool:
			*d = val != 0
		case *any:
			*d = val
		case *sql.NullInt64:
			d.Int64 = val
			d.Valid = true
		case *sql.NullInt32:
			if val < math.MinInt32 || val > math.MaxInt32 {
				return fmt.Errorf("integer overflow: %d does not fit in int32", val)
			}
			d.Int32 = int32(val)
			d.Valid = true
		case *sql.NullInt16:
			if val < math.MinInt16 || val > math.MaxInt16 {
				return fmt.Errorf("integer overflow: %d does not fit in int16", val)
			}
			d.Int16 = int16(val)
			d.Valid = true
		case *sql.NullBool:
			d.Bool = val != 0
			d.Valid = true
		default:
			if scanner, ok := dest.(sql.Scanner); ok {
				return scanner.Scan(val)
			}
			return fmt.Errorf("cannot scan INTEGER into %T", dest)
		}

	case sqlite.TypeFloat:
		val := stmt.ColumnFloat(col)
		switch d := dest.(type) {
		case *float32:
			*d = float32(val)
		case *float64:
			*d = val
		case *any:
			*d = val
		case *sql.NullFloat64:
			d.Float64 = val
			d.Valid = true
		default:
			if scanner, ok := dest.(sql.Scanner); ok {
				return scanner.Scan(val)
			}
			return fmt.Errorf("cannot scan FLOAT into %T", dest)
		}

	case sqlite.TypeText:
		val := stmt.ColumnText(col)
		switch d := dest.(type) {
		case *string:
			*d = val
		case *[]byte:
			*d = []byte(val)
		case *any:
			*d = val
		case *sql.NullString:
			d.String = val
			d.Valid = true
		default:
			if scanner, ok := dest.(sql.Scanner); ok {
				return scanner.Scan(val)
			}
			return fmt.Errorf("cannot scan TEXT into %T", dest)
		}

	case sqlite.TypeBlob:
		// Get the size of the blob first
		size := stmt.ColumnLen(col)
		buf := make([]byte, size)
		stmt.ColumnBytes(col, buf)

		switch d := dest.(type) {
		case *[]byte:
			*d = buf
		case *string:
			*d = string(buf)
		case *any:
			*d = buf
		default:
			if scanner, ok := dest.(sql.Scanner); ok {
				return scanner.Scan(buf)
			}
			return fmt.Errorf("cannot scan BLOB into %T", dest)
		}

	default:
		return fmt.Errorf("unknown column type: %v", colType)
	}

	return nil
}
