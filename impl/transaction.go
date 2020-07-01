package impl

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// Transaction executes txFunc within a database transaction that is passed in as tx sqldb.Connection.
// The transaction will be rolled back if txFunc returns an error or panics.
// Recovered panics are re-paniced after the transaction is rolled back.
// Rollback errors are logged with sqldb.ErrLogger.
// Transaction returns all errors from txFunc or transaction commit errors happening after txFunc.
// If conn is already a transaction, then txFunc is executed within this transaction
// ignoring opts and without calling another Begin or Commit in this Transaction call.
// Errors or panics will roll back the inherited transaction though.
func Transaction(ctx context.Context, opts *sql.TxOptions, conn sqldb.Connection, txFunc func(tx sqldb.Connection) error) (err error) {
	var tx sqldb.Connection
	if conn.IsTransaction() {
		tx = conn
		if ctx.Err() != nil {
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				sqldb.ErrLogger.Printf("sqldb.Transaction error (%s) from rollback after context error: %s", e, ctx.Err())
			}
			return ctx.Err()
		}
	} else {
		tx, err = conn.Begin(ctx, opts)
		if err != nil {
			return fmt.Errorf("sqldb.Transaction Begin returned: %w", err)
		}
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

		if conn.IsTransaction() {
			return
		}

		e := tx.Commit()
		if e != nil {
			// Set Commit error as function return value
			err = fmt.Errorf("sqldb.Transaction commit: %w", e)
		}
	}()

	return txFunc(tx)
}
