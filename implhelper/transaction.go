package implhelper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sqldb "github.com/domonda/go-sqldb"
)

// Transaction executes txFunc within a transaction that is passed in as tx Connection.
// The transaction will be rolled back if txFunc returns an error or panics.
func Transaction(ctx context.Context, opts *sql.TxOptions, conn sqldb.Connection, txFunc func(tx sqldb.Connection) error) (err error) {
	tx, e := conn.Begin(ctx, opts) // use e to keep err accessible in defer func below
	if e != nil {
		return fmt.Errorf("sqldb.Transaction begin: %w", e)
	}

	defer func() {
		if r := recover(); r != nil {
			// txFunc paniced
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, log e so it doesn't get lost
				sqldb.ErrLogger.Printf("sqldb.Transaction error %s from rollback after panic: %+v", e, r)
			}
			panic(r) // re-throw panic after Rollback
		}

		if err != nil {
			// txFunc returned an error
			e := tx.Rollback()
			if e != nil && !errors.Is(e, sql.ErrTxDone) {
				// Double error situation, wrap err with e so it doesn't get lost
				err = fmt.Errorf("sqldb.Transaction error %s from rollback after error: %w", e, err)
			}
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
