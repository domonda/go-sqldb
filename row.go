package sqldb

import (
	"context"
	"database/sql"
	"errors"
)

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

///////////////////////////////////////////////////////////////////////////////

// RowWithError returns a dummy Row
// where all methods return the passed error.
func RowWithError(err error) Row {
	return errRow{err}
}

type errRow struct{ err error }

func (e errRow) Columns() ([]string, error) { return nil, e.err }
func (e errRow) Scan(dest ...any) error     { return e.err }

///////////////////////////////////////////////////////////////////////////////

type sqlRow struct {
	ctx   context.Context // ctx is checked for every row and passed through to callbacks
	rows  *sql.Rows
	conn  Connection // for error wrapping
	query string     // for error wrapping
	args  []any      // for error wrapping
}

func NewRow(ctx context.Context, rows *sql.Rows, conn Connection, query string, args []any) Row {
	return &sqlRow{ctx, rows, conn, query, args}
}

func (r *sqlRow) Columns() ([]string, error) {
	columns, err := r.rows.Columns()
	if err != nil {
		return nil, WrapErrorWithQuery(err, r.query, r.args, r.conn.Config().ParamPlaceholderFormatter)
	}
	return columns, nil
}

func (r *sqlRow) Scan(dest ...any) (err error) {
	defer func() {
		err = combineTwoErrors(err, r.rows.Close())
		if err != nil {
			err = WrapErrorWithQuery(err, r.query, r.args, r.conn.Config().ParamPlaceholderFormatter)
		}
	}()

	if r.ctx.Err() != nil {
		return r.ctx.Err()
	}

	// TODO(bradfitz): for now we need to defensively clone all
	// []byte that the driver returned (not permitting
	// *RawBytes in Rows.Scan), since we're about to close
	// the Rows in our defer, when we return from this function.
	// the contract with the driver.Next(...) interface is that it
	// can return slices into read-only temporary memory that's
	// only valid until the next Scan/Close. But the TODO is that
	// for a lot of drivers, this copy will be unnecessary. We
	// should provide an optional interface for drivers to
	// implement to say, "don't worry, the []bytes that I return
	// from Next will not be modified again." (for instance, if
	// they were obtained from the network anyway) But for now we
	// don't care.
	for _, dp := range dest {
		if _, ok := dp.(*sql.RawBytes); ok {
			return errors.New("sql: RawBytes isn't allowed on Row.Scan")
		}
	}
	if !r.rows.Next() {
		if err := r.rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	err = r.rows.Scan(dest...)
	if err != nil {
		return err
	}
	return r.rows.Close()
}
