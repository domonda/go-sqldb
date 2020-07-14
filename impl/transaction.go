package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// Transaction executes txFunc within a database transaction that is passed in to txFunc as tx sqldb.Connection.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If parentConn is already a transaction, then it is passed through to txFunc unchanged as tx sqldb.Connection
// and no parentConn.Begin, Commit, or Rollback calls will occour within this Transaction call.
// Errors and panics from txFunc will rollback the transaction if parentConn was not already a transaction.
// Recovered panics are re-paniced and rollback errors after a panic are logged with sqldb.ErrLogger.
func Transaction(ctx context.Context, opts *sql.TxOptions, parentConn sqldb.Connection, txFunc func(tx sqldb.Connection) error) (err error) {
	if parentConn.IsTransaction() {
		// Check context for error because there is no
		// Begin(ctx) call that would return the error
		if ctx.Err() != nil {
			return ctx.Err()
		}

		return txFunc(parentConn)
	}

	tx, e := parentConn.Begin(ctx, opts)
	if e != nil {
		return fmt.Errorf("sqldb.Transaction Begin error: %w", e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				sqldb.ErrLogger.Printf("sqldb.Transaction error (%s) from rollback after panic: %+v", e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("sqldb.Transaction error (%s) from rollback after error: %w", e, err)
			}
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("sqldb.Transaction Commit error: %w", e)
		}
	}()

	return txFunc(tx)
}
